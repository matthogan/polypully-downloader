package http

import (
	"container/list"
	"context"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"sync"
	"time"

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
			Id:               id,
			Uri:              uri,
			Destination:      viper.GetString("download_directory"),
			PathTemplate:     viper.GetString("path_template"),
			MaxConcFragments: viper.GetInt("max_conc_fragments"),
			MaxFragmentSz:    viper.GetInt("max_fragment_size"),
			MinFragmentSz:    viper.GetInt("min_fragment_size"),
			Retries:          viper.GetInt("retries"),
			FileMode:         fs.FileMode(viper.GetUint32("filemode")),
			BufferSize:       viper.GetInt("buffer_size"),
			Errors:           list.New(),
			Fragments:        make(map[int]*model.Fragment),
			FragLock:         &sync.RWMutex{},
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
	d.StartTime = time.Now()
	if err := d.Validate(); err != nil {
		slog.Error("validate", "error", err)
		d.Status = model.DownloadError
		return err
	}
	filename := path.Base(d.Uri)
	dir := d.PathTemplate
	if dir != "" {
		dir = fmt.Sprintf(dir, filename, d.Id)
	}
	if err := d.BurnDirectory(dir); err != nil {
		d.Status = model.DownloadError
		return fmt.Errorf("burn directory: %w", err)
	}
	size, err := d.GetFileSize()
	if err != nil {
		slog.Info("file size", "error", err) // content-length is not always present
		err = nil
	}
	d.FileSize = int(size)
	d.File = d.Fqfn(d.Destination, dir, filename) // fqfn
	slog.Debug("download", "filename", d.File)

	d.Fragments = d.fragments()
	slog.Debug("download", "fragments", len(d.Fragments))

	go d.downloadRoutine()
	return err
}

// calculate the fragments based on the max concurrent downloads
// and fragment size configuration parameters.
func (d *Download) fragments() map[int]*model.Fragment {
	fragments := make(map[int]*model.Fragment)
	fragmentSize := d.MaxFragmentSz
	nFragments := int(d.FileSize/fragmentSize) + 1
	if d.FileSize <= d.MinFragmentSz {
		d.MaxConcFragments = 1
		nFragments = 1
		fragmentSize = d.FileSize
	} else if d.FileSize < d.MaxFragmentSz {
		fragmentSize = int(d.FileSize / (d.MaxConcFragments - 1))
		nFragments = int(d.FileSize/fragmentSize) + 1
	}
	// create the fragments
	// last one will be an odd size
	for i := 0; i < nFragments; i++ {
		start := i * fragmentSize
		end := start + fragmentSize - 1
		if i == nFragments-1 {
			end = d.FileSize - 1
		}
		fragments[i] = &model.Fragment{
			Index:    int(i),
			Start:    start,
			End:      end, // -1 possibly
			Filename: d.File + "." + strconv.FormatInt(int64(i), 10),
		}
	}
	return fragments
}
