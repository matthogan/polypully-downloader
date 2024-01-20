package http

import (
	"container/list"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"

	apperrors "github.com/codejago/polypully/downloader/internal/app/errors"
)

type CommunicationClient interface {
	FetchData(d Download, fragment Fragment) error
}

type Download struct {
	Id            string
	Client        CommunicationClient
	File          *os.File
	Uri           string
	Destination   string
	Fragments     int
	MinFragmentSz int
	Retries       int
	FileMode      fs.FileMode
	Status        DownloadStatus
	Errors        list.List
	Context       context.Context
	Cancel        func()
	BufferSize    int
}

type Fragment struct {
	Destination io.Writer
	Start       int64
	End         int64
	Error       error
}

type DownloadStatus int32

const (
	DownloadUndefined DownloadStatus = iota
	DownloadRunning
	DownloadComplete
	DownloadError
)

// String method is automatically called when we try to print the value of the DownloadStatus
func (d DownloadStatus) String() string {
	return [...]string{"undefined", "running", "complete", "error"}[d]
}

type ContextKey string

func (d *Download) downloadRoutine(fragmentSize int64, filename string, size int64) {

	if err := d.InitializeFile(filename); err != nil {
		d.Status = DownloadError
		d.Errors.PushFront(err)
		slog.Error("failed in initialize %s", d.Status)
		return
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
				d.Errors.PushFront(err)
				slog.Error("failed in download %s", DownloadError)
			}
		}

		if err := d.MergeFiles(filename); err != nil {
			slog.Debug("merge", "filename", filename, "error", err)
			if r < d.Retries {
				slog.Info("retry", "filename", filename, "retry", r, "retries", d.Retries)
				continue
			}
			d.Status = DownloadError
			d.Errors.PushFront(err)
			slog.Error("failed in merge %s", DownloadError)
		}

		slog.Info("complete", "filename", filename)
		d.Status = DownloadComplete
	}
}

// cleanup in case of error
func (d *Download) CancelDownload(filename string) {
	d.Cancel()
	if d.Status == DownloadError {
		if err := os.Remove(filename); err != nil {
			slog.Error("remove failed", "filename", filename, "error", err)
		}
	}
}

func (d *Download) Validate() error {
	if d.Destination == "" {
		return &apperrors.ValidationError{Msg: "destination not set"}
	}
	if d.Fragments == 0 {
		return &apperrors.ValidationError{Msg: "fragments not set"}
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

func (d *Download) InitializeFile(filename string) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, d.FileMode)
	if err != nil {
		return err
	}
	if err := file.Truncate(0); err != nil {
		return err
	}
	d.File = file
	return nil
}

func (d *Download) InitializeFragmentFile(filename string) (*os.File, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, d.FileMode)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (d *Download) MergeFile(fragmentFilename string) error {
	fragmentFile, err := os.Open(fragmentFilename)
	if err != nil {
		return err
	}
	defer fragmentFile.Close()
	// end of the file
	_, err = d.File.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	// append the data
	_, err = io.Copy(d.File, fragmentFile)
	return err
}

func (d *Download) GetFragmentFilename(filename string, i int64) string {
	return filename + "." + strconv.FormatInt(i, 10)
}

func (d *Download) MergeFiles(filename string) error {
	for i := int64(0); i < int64(d.Fragments); i++ {
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
	errChan := make(chan error, d.Fragments)
	var wg sync.WaitGroup

	for i := int64(0); i < int64(d.Fragments); i++ {
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
	if i == int64(d.Fragments)-1 {
		end = size - 1
	}
	fragmentFilename := d.GetFragmentFilename(filename, i)
	file, err := d.InitializeFragmentFile(fragmentFilename)
	if err != nil {
		slog.Error("initialize", "fragmentFilename", fragmentFilename, "error", err)
	}
	defer file.Close()
	fragment := Fragment{
		Destination: file,
		Start:       start,
		End:         end, // -1 possibly
	}
	return d.Client.FetchData(*d, fragment)
}
