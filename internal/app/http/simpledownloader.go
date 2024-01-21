package http

import (
	"container/list"
	"context"
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
		Id: id,
		Client: NewHttpClient(&HttpClientConfig{
			Timeout:   viper.GetDuration("timeout"),
			Redirects: viper.GetInt("redirects")}),
		Uri:           uri,
		Destination:   viper.GetString("download_directory"),
		MaxFragments:  viper.GetInt64("max_fragments"),
		MinFragmentSz: viper.GetInt64("min_fragment_size"),
		Retries:       viper.GetInt64("retries"),
		FileMode:      fs.FileMode(viper.GetUint32("filemode")),
		Context:       ctx,
		BufferSize:    viper.GetInt64("buffer_size"),
		Errors:        list.New(),
		Cancel: func() {
			slog.Info("cancelling", "id", ctx.Value(ContextKey("download_id")))
			cancel()
		},
		Fragments: make(map[int]*Fragment),
		fragLock:  &sync.RWMutex{},
	}
}

func (d *Download) Download() error {
	d.Status = DownloadInitialising
	if err := d.Validate(); err != nil {
		slog.Error("validate", "error", err)
		d.Status = DownloadError
		return err
	}

	filename := d.GetFilename() // fqfn
	slog.Debug("download", "filename", filename)
	size, err := d.GetFileSize()
	if err != nil {
		slog.Info("file size", "error", err) // content-length is not always present
		err = nil
	}
	d.FileSize = size

	fragmentSize := size
	if size <= int64(d.MinFragmentSz) { // may be 0
		d.MaxFragments = 1
	} else {
		fragmentSize = size / int64(d.MaxFragments)
	}
	slog.Debug("download", "fragmentSize", fragmentSize)

	go d.downloadRoutine(fragmentSize, filename, size)

	return err
}
