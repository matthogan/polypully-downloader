package http

import (
	"container/list"
	"context"
	"io/fs"
	"sync"

	model "github.com/codejago/polypully/downloader/internal/app/model"
	"github.com/codejago/polypully/downloader/internal/app/storage"
	appevents "github.com/matthogan/polypully-events"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"golang.org/x/exp/slog"
)

// ContextKey is a type for the context key in the Context
type ContextKey string

// Download represents a download and is cancellable
func NewDownload(uri string, events appevents.EventsApi, storage storage.StorageApi) Download {
	id := uuid.New().String()
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, ContextKey("download_id"), id)
	return Download{ // struct
		Resource: model.Resource{
			Id:            id,
			Uri:           uri,
			Destination:   viper.GetString("download_directory"),
			MaxFragments:  viper.GetInt64("max_fragments"),
			MinFragmentSz: viper.GetInt64("min_fragment_size"),
			Retries:       viper.GetInt64("retries"),
			FileMode:      fs.FileMode(viper.GetUint32("filemode")),
			BufferSize:    viper.GetInt64("buffer_size"),
			Errors:        list.New(),
			Fragments:     make(map[int]*model.Fragment),
			FragLock:      &sync.RWMutex{},
		},
		Client: NewHttpClient(&HttpClientConfig{
			Timeout:   viper.GetDuration("timeout"),
			Redirects: viper.GetInt("redirects")}),
		Context: ctx,
		Cancel: func() {
			slog.Info("cancelling", "id", ctx.Value(ContextKey("download_id")))
			cancel()
		},
		Events:  events,
		storage: storage,
	}
}

func (d *Download) Download() error {

	d.Status = model.DownloadInitialising
	if err := d.Validate(); err != nil {
		slog.Error("validate", "error", err)
		d.Status = model.DownloadError
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
