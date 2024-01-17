package http

import (
	"context"
	"errors"
	"io/fs"

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
		Id:            id,
		Client:        NewHttpClient(),
		Uri:           uri,
		Destination:   viper.GetString("download_directory"),
		Fragments:     viper.GetInt("fragments"),
		MinFragmentSz: viper.GetInt("min_fragment_size"),
		Retries:       viper.GetInt("retries"),
		FileMode:      fs.FileMode(viper.GetUint32("filemode")),
		Context:       ctx,
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
		slog.Error("file size", "error", err) // content-length is not always present
	}

	fragmentSize := size
	if size <= int64(d.MinFragmentSz) { // may be 0
		d.Fragments = 1
	} else {
		fragmentSize = size / int64(d.Fragments)
	}
	slog.Info("download", "fragmentSize", fragmentSize)

	if err = d.InitializeFile(filename); err != nil {
		slog.Error("initialize", "filename", filename, "error", err)
		return err
	}
	defer d.File.Close()

	for r := 0; r <= d.Retries; r++ {
		errorChannel := d.DownloadFragments(fragmentSize, filename, size)
		for err := range errorChannel {
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

		if err = d.MergeFiles(filename); err != nil {
			slog.Error("merge", "filename", filename, "error", err)
			if r < d.Retries {
				slog.Info("retry", "filename", filename, "retry", r, "retries", d.Retries)
				continue
			}
			d.Status = DownloadError
			return errors.New("failed in merge")
		}

		slog.Info("complete", "filename", filename)
		d.Status = DownloadComplete
		return nil
	}

	return nil
}
