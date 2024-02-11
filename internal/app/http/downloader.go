package http

import (
	"context"
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

func (d *Download) downloadRoutine(fragmentSize int64, filename string, size int64) {

	d.Status = model.DownloadRunning

	if err := d.UpdateResource(); err != nil {
		return
	}

	if d.Status != model.DownloadError {
		d.download(fragmentSize, filename, size)
	}

	if d.Status == model.DownloadRunning {
		d.Status = model.DownloadComplete
	}

	d.UpdateResource()
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

func (d *Download) download(fragmentSize int64, filename string, size int64) {

	for r := int64(0); r <= d.Retries; r++ {

		if err := d.InitializeFile(filename); err != nil {
			d.Status = model.DownloadInitError
			d.Errors.PushFront(err)
			slog.Error("failed in initialize %s", d.Status)
			break
		}

		errorChannel := d.DownloadFragments(fragmentSize, filename, size)
		for err := range errorChannel {
			if err != nil {
				slog.Error("download", "filename", filename, "error", err)
				if r < d.Retries {
					slog.Info("retry", "filename", filename, "retry", r, "retries", d.Retries)
					continue
				}
				d.Status = model.DownloadError
				d.Errors.PushFront(err)
				slog.Error("failed in download %s", model.DownloadError)
			}
		}

		if err := d.MergeFiles(filename); err != nil {
			slog.Debug("merge", "filename", filename, "error", err)
			if r < d.Retries {
				slog.Info("retry", "filename", filename, "retry", r, "retries", d.Retries)
				continue
			}
			d.Status = model.DownloadError
			d.Errors.PushFront(err)
			slog.Error("failed in merge %s", model.DownloadError)
		}

		slog.Info("complete", "filename", filename)
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
	if d.MaxFragments == 0 {
		return &apperrors.ValidationError{Msg: "max fragments not set"}
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

func (d *Download) GetFilename() string {
	filename := path.Base(d.Uri)
	path := path.Join(d.Destination, filename)
	// if the file exists, append a number to the filename of the existing file
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
func (d *Download) InitializeFile(filename string) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, d.FileMode)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Truncate(0); err != nil {
		return err
	}
	d.File = filename
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

func (d *Download) GetFragmentFilename(filename string, i int64) string {
	return filename + "." + strconv.FormatInt(i, 10)
}

func (d *Download) MergeFiles(filename string) error {
	for i := int64(0); i < int64(d.MaxFragments); i++ {
		fragment := d.GetFragmentFilename(filename, i)
		err := d.MergeFile(fragment)
		if err != nil {
			return err
		}
		err = os.Remove(fragment)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Download) DownloadFragments(fragmentSize int64, filename string, size int64) chan error {
	errChan := make(chan error, d.MaxFragments)
	var wg sync.WaitGroup

	for i := int64(0); i < d.MaxFragments; i++ {
		wg.Add(1)
		go func(i int64) {
			defer wg.Done()
			err := d.DownloadSingleFragment(i, fragmentSize, filename, size)
			if err != nil {
				errChan <- err
				d.CancelDownload(filename)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)
	return errChan
}

func (d *Download) DownloadSingleFragment(i, fragmentSize int64, filename string, size int64) error {
	start := i * fragmentSize
	end := start + fragmentSize - 1
	if i == d.MaxFragments-1 {
		end = size - 1
	}
	fragmentFilename := d.GetFragmentFilename(filename, i)
	file, err := d.InitializeFragmentFile(fragmentFilename)
	if err != nil {
		slog.Error("initialize", "fragmentFilename", fragmentFilename, "error", err)
	}
	defer file.Close()
	fragment := &model.Fragment{
		Index:       int(i),
		Destination: file,
		Start:       start,
		End:         end, // -1 possibly
		StartTime:   time.Now(),
	}
	d.setFragment(i, fragment)
	err = d.Client.FetchData(d.Context, &d.Resource, fragment) // download through the configured channel
	if err != nil {
		slog.Error("fetch", "fragmentFilename", fragmentFilename, "error", err)
	}
	fragment.EndTime = time.Now()
	return err
}

// setFragment is a thread-safe way to write to the map
// minimizes the time the map is locked
func (d *Download) setFragment(i int64, fragment *model.Fragment) {
	d.FragLock.Lock() // blocks other readers and writers
	defer d.FragLock.Unlock()
	d.Fragments[int(i)] = fragment
}
