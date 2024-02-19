package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	apperrors "github.com/codejago/polypully/downloader/internal/app/errors"
	model "github.com/codejago/polypully/downloader/internal/app/model"
	"github.com/codejago/polypully/downloader/internal/app/storage"
	appevents "github.com/matthogan/polypully-events"
)

// Download represents a download with some runtime aspects
type Download struct {
	// Generic resource definition
	model.Resource
	// Client is the communication client
	Client model.CommunicationClient
	// Context is used to cancel the download
	Context context.Context
	// Cancel is a function that can be called to cancel the download
	Cancel func()
	// Events update the wider system
	Events appevents.EventsApi
	// Local storage for the download
	storage storage.StorageApi
}

func (d *Download) downloadRoutine() {

	d.Status = model.DownloadRunning

	if err := d.UpdateResource(); err != nil {
		return
	}

	if d.Status != model.DownloadError {
		d.download()
	}

	if d.Status == model.DownloadRunning {
		d.Status = model.DownloadComplete
		d.EndTime = time.Now()
		if err := d.finalize(); err != nil {
			d.Status = model.DownloadError
			d.Errors.PushFront(err)
		}
	}
}

func (d *Download) finalize() error {
	if err := d.CreateManifest(); err != nil {
		return fmt.Errorf("failed to create manifest: %v", err)
	}
	if err := d.UpdateResource(); err != nil {
		return fmt.Errorf("failed to update resource: %v", err)
	}
	return nil
}

func (d *Download) UpdateResource() error {
	if d.Status == model.DownloadError {
		return nil
	}
	if err := d.storage.UpdateResource(&d.Resource); err != nil {
		d.Errors.PushFront(err)
		d.Status = model.DownloadError
		return err
	}
	if err := d.Events.Notify(appevents.NewDownloadEvent(d.Status.String(), d.Id)); err != nil {
		d.Errors.PushFront(err)
		d.Status = model.DownloadError
		return err
	}
	return nil
}

func (d *Download) download() {

	for r := 0; r <= d.Retries; r++ {

		if err := d.InitializeFile(); err != nil {
			d.Status = model.DownloadInitError
			d.Errors.PushFront(err)
			slog.Error("failed in initialize %s", d.Status)
			break
		}

		errorChannel := d.DownloadFragments()
		for err := range errorChannel {
			if err != nil {
				slog.Error("download", "filename", d.File, "error", err)
				if r < d.Retries {
					slog.Info("retry", "filename", d.File, "retry", r, "retries", d.Retries)
					continue
				}
				d.Status = model.DownloadError
				d.Errors.PushFront(err)
				slog.Error("failed in download %s", model.DownloadError)
			}
		}

		if err := d.MergeFiles(d.File); err != nil {
			slog.Debug("merge", "filename", d.File, "error", err)
			if r < d.Retries {
				slog.Info("retry", "filename", d.File, "retry", r, "retries", d.Retries)
				continue
			}
			d.Status = model.DownloadError
			d.Errors.PushFront(err)
			slog.Error("failed in merge %s", model.DownloadError)
		}

		slog.Info("complete", "filename", d.File)
		break // success or failure
	}
}

// cleanup in case of error
func (d *Download) CancelDownload(filename string) {
	d.Cancel()
	if d.Status == model.DownloadError {
		if err := os.Remove(filename); err != nil {
			slog.Error("remove failed", "filename", filename, "error", err)
		}
	}
}

func (d *Download) Validate() error {
	if d.Destination == "" {
		return &apperrors.ValidationError{Msg: "destination not set"}
	}
	if d.PathTemplate == "" {
		return &apperrors.ValidationError{Msg: "path template not set"}
	}
	if d.MaxConcFragments == 0 {
		return &apperrors.ValidationError{Msg: "max concurrent fragments not set"}
	}
	if d.Uri == "" {
		return &apperrors.ValidationError{Msg: "uri not set"}
	}
	if d.FileMode == 0 {
		return &apperrors.ValidationError{Msg: "filemode not set"}
	}
	return nil
}

func (d *Download) GetFileSize() (int64, error) {
	resp, err := http.Head(d.Uri)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("http request error: %s", resp.Status)
	}
	size, err := strconv.ParseInt(resp.Header.Get("content-length"), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse content-length header: %v", err)
	}
	return size, nil
}

func (d *Download) BurnDirectory(structure string) error {
	path := path.Join(d.Destination, structure)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(path, 0755)
		}
		if err != nil {
			slog.Error("mkdir", "path", path, "error", err)
		}
		return err
	}
	return nil
}

// if the file exists, append a number to the filename of the existing file
func (d *Download) Fqfn(root string, directories string, filename string) string {
	path := path.Join(root, directories, filename)
	if _, err := os.Stat(path); err == nil {
		i := 1
		for {
			newPath := fmt.Sprintf("%s.%d", path, i)
			if _, err := os.Stat(newPath); err == nil {
				i++
				continue
			}
			path = newPath
			break
		}
	}
	return path
}

// InitializeFile creates a file and truncates it to 0
// Keeps the filename in the struct
// Closes the file
func (d *Download) InitializeFile() error {
	file, err := os.OpenFile(d.File, os.O_CREATE|os.O_WRONLY, d.FileMode)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Truncate(0); err != nil {
		return err
	}
	return nil
}

func (d *Download) InitializeFragmentFile(filename string) (*os.File, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, d.FileMode)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// MergeFile appends the fragment file to the main file.
// Deletes the fragment file.
// The main file is a shared resource. The caller manages
// any locking semantics.
func (d *Download) MergeFile(fragmentFilename string) error {
	// read from the fragment file
	fragmentFile, err := os.OpenFile(fragmentFilename, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open fragment file: %v", err)
	}
	defer fragmentFile.Close()
	// write to the main file
	file, err := os.OpenFile(d.File, os.O_WRONLY, d.FileMode)
	if err != nil {
		return fmt.Errorf("failed to open main file: %v", err)
	}
	defer file.Close()
	// end of the file
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek to end of file: %v", err)
	}
	// append the data
	_, err = io.Copy(file, fragmentFile)
	if err != nil {
		return fmt.Errorf("failed to copy fragment to main file: %v", err)
	}
	return err
}

func (d *Download) MergeFiles(filename string) error {
	for i := 0; i < len(d.Fragments); i++ {
		f := d.Fragments[i]
		if err := d.MergeFile(f.Filename); err != nil {
			return err
		}
		if err := os.Remove(f.Filename); err != nil {
			return err
		}
	}
	return nil
}

// download the fragments concurrently using channels rather than
// a latch or a semaphore.

func (d *Download) DownloadFragments() chan error {
	var wg sync.WaitGroup                                      // wait for all fragments to download
	errChan := make(chan error, len(d.Fragments))              // collect errors
	fragChan := make(chan *model.Fragment, d.MaxConcFragments) // control concurrent downloads
	// push onto the channel
	// push onto the channel
	go func() {
		for _, f := range d.Fragments {
			fragChan <- f
		}
		close(fragChan) // close the channel here, after all fragments have been sent
	}()
	// pop off the channel
	for f := range fragChan {
		wg.Add(1)
		go func(f *model.Fragment) {
			defer wg.Done()
			if err := d.DownloadSingleFragment(f); err != nil {
				errChan <- err
				d.CancelDownload(d.File)
			}
		}(f)
	}
	wg.Wait()
	close(errChan)
	return errChan
}

func (d *Download) DownloadSingleFragment(f *model.Fragment) error {
	file, err := d.InitializeFragmentFile(f.Filename)
	if err != nil {
		slog.Error("initialize", "fragmentFilename", f.Filename, "error", err)
	}
	defer file.Close()
	f.StartTime = time.Now()
	f.Destination = file
	// download through the configured channel
	if err = d.Client.FetchData(d.Context, &d.Resource, f); err != nil {
		slog.Error("fetch", "fragmentFilename", f.Filename, "error", err)
	}
	f.EndTime = time.Now()
	return err
}

// Create a manifest of the download alongside
// the file.
func (d *Download) CreateManifest() error {
	data, err := json.MarshalIndent(d.Resource, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %v", err)
	}
	manifest := d.Fqfn(path.Dir(d.File), "", "manifest.mf")
	file, err := os.OpenFile(manifest, os.O_CREATE|os.O_WRONLY, d.FileMode)
	if err != nil {
		return fmt.Errorf("failed to open manifest file: %v", err)
	}
	defer file.Close()
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write manifest file: %v", err)
	}
	slog.Debug("manifest", "written", manifest)
	return nil
}
