package http

import (
	"context"
	"errors"
	"io/fs"
	"sync"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
)

// Download represents a download and is cancellable
func NewDownload(uri string) Download {
	id := uuid.New().String()
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, ContextKey("download_id"), id)
	return Download{
		Id:          id,
		Client:      NewHttpClient(),
		Uri:         uri,
		Destination: viper.GetString("download_directory"),
		Fragments:   viper.GetInt("fragments"),
		Retries:     viper.GetInt("retries"),
		FileMode:    fs.FileMode(viper.GetUint32("filemode")),
		Context:     ctx,
		Cancel: func() {
			slog.Info("cancelling", "id", ctx.Value(ContextKey("download_id")))
			cancel()
		},
		BufferSize: viper.GetInt("buffer_size"),
	}
}

func (d *Download) Download() error {
	d.Status = DownloadRunning
	if err := d.Validate(); err != nil {
		slog.Error("validate", "error", err)
		d.Status = DownloadError
		return err
	}

	filename := d.GetFilename() // fqfn
	slog.Info("download", "filename", filename)
	size, err := d.GetFileSize()
	if err != nil {
		slog.Error("file size", "error", err)
		d.Status = DownloadError
		return err
	}

	fragmentSize := size
	if size < int64(d.Fragments) {
		d.Fragments = 1
	} else {
		fragmentSize = size / int64(d.Fragments)
	}
	slog.Info("download", "fragmentSize", fragmentSize)

	err = d.InitializeFile(filename, size)
	if err != nil {
		slog.Error("initialize", "filename", filename, "error", err)
	}
	defer d.File.Close()

	for r := 0; r <= d.Retries; r++ {
		errChan := make(chan error, d.Fragments)
		var wg sync.WaitGroup

		for i := int64(0); i < int64(d.Fragments); i++ {
			wg.Add(1)
			go func(i int64) {
				defer wg.Done()
				err := d.downloadSingleFragment(i, fragmentSize, filename, size)
				if err != nil {
					errChan <- err
					d.CancelDownload(filename)
				}
			}(i)
		}

		wg.Wait()
		close(errChan)

		for err := range errChan {
			if err != nil {
				slog.Error("download", "filename", filename, "error", err)
				if r < d.Retries {
					slog.Info("retry", "filename", filename, "retry", r, "retries", d.Retries)
					continue
				}
				d.Status = DownloadError
				return errors.New("failed in download")
			}
		}

		for i := int64(0); i < int64(d.Fragments); i++ {
			err := d.MergeFile(d.GetFragmentFilename(filename, i))
			if err != nil {
				slog.Error("merge", "filename", filename, "error", err)
				if r < d.Retries {
					slog.Info("retry", "filename", filename, "retry", r, "retries", d.Retries)
					continue
				}
				d.Status = DownloadError
				return errors.New("failed in merge")
			}
		}

		slog.Info("complete", "filename", filename)
		d.Status = DownloadComplete
		return nil
	}

	return nil
}

func (d *Download) downloadSingleFragment(i, fragmentSize int64, filename string, size int64) error {
	start := i * fragmentSize
	end := start + fragmentSize - 1
	if i == int64(d.Fragments)-1 {
		end = size - 1
	}
	fragmentFilename := d.GetFragmentFilename(filename, i)
	file, err := d.InitializeFragmentFile(fragmentFilename, end-start)
	if err != nil {
		slog.Error("initialize", "fragmentFilename", fragmentFilename, "error", err)
	}
	defer file.Close()
	fragment := Fragment{
		Destination: file,
		Start:       start,
		End:         end,
	}
	return d.Client.FetchData(*d, fragment)
}
