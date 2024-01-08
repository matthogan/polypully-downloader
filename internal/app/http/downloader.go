package http

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strconv"

	apperrors "github.com/codejago/polypully/downloader/internal/app/errors"
)

type CommunicationClient interface {
	FetchData(d Download, fragment Fragment) error
}

type Download struct {
	Id          string
	Client      CommunicationClient
	File        *os.File
	Uri         string
	Destination string
	Fragments   int
	Retries     int
	FileMode    fs.FileMode
	Status      DownloadStatus
	Error       error
	Context     context.Context
	Cancel      func()
	BufferSize  int
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

type ContextKey string

// cleanup in case of error
func (d *Download) CancelDownload(filename string) {
	d.Cancel()
	if d.Status == DownloadError {
		os.Remove(filename)
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
	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
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

func (d *Download) InitializeFile(filename string, size int64) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, d.FileMode)
	if err != nil {
		return err
	}
	if err := file.Truncate(size); err != nil {
		return err
	}
	d.File = file
	return nil
}

func (d *Download) InitializeFragmentFile(filename string, size int64) (*os.File, error) {
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
	_, err = io.Copy(d.File, fragmentFile)
	return err
}

func (d *Download) GetFragmentFilename(filename string, i int64) string {
	return filename + "." + strconv.FormatInt(i, 10)
}
