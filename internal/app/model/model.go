package model

// Shared data structures and interfaces

import (
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sync"
	"time"
)

// DownloadStatus alias represents the status of the download
type DownloadStatus int32

// DownloadStatus represents the status of the download
// APIs should represent this value as a string
const (
	DownloadUndefined DownloadStatus = iota
	DownloadInitialising
	DownloadRunning
	DownloadComplete
	DownloadError
	DownloadInitError
)

// String method is automatically called when we try to print the value of the model.DownloadStatus
func (d DownloadStatus) String() string {
	return [...]string{"undefined", "initializing", "running", "complete", "error", "init_error"}[d]
}

// CommunicationClient is an interface for fetching a fragment of data
type CommunicationClient interface {
	FetchData(context context.Context, d *Resource, fragment *Fragment) error
}

// Fragment represents a part of the download
// When restarting fragments from storage, the Destination
// must be recreated.
type Fragment struct {
	Index       int       `json:"index"`
	Destination io.Writer `json:"-"`
	Start       int64     `json:"start"`
	End         int64     `json:"end"`
	Error       error     `json:"error"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Progress    int64     `json:"progress"`
}

// Central data structure for the download
// dependency on the "appevents" package
type Resource struct {
	Id            string            `json:"id"`
	File          string            `json:"file"`
	Uri           string            `json:"uri"`
	Destination   string            `json:"destination"`
	MaxFragments  int64             `json:"max_fragments"`
	MinFragmentSz int64             `json:"min_fragment_size"`
	Retries       int64             `json:"retries"`
	FileMode      fs.FileMode       `json:"filemode"`
	Status        DownloadStatus    `json:"status"`
	Errors        *list.List        `json:"errors"`
	BufferSize    int64             `json:"buffer_size"`
	Fragments     map[int]*Fragment `json:"fragments"`
	FileSize      int64             `json:"file_size"`
	FragLock      *sync.RWMutex     `json:"-"` // FragLock is a lock for the Fragments map
}

func (r Resource) Identifier() string {
	return r.Id
}

// Custom JSON marshallers

func (f *Fragment) MarshalJSON() ([]byte, error) {
	type Alias Fragment
	alias := struct {
		*Alias
		Error string `json:"error"`
	}{
		Alias: (*Alias)(f),
		Error: "",
	}
	if f.Error != nil {
		alias.Error = f.Error.Error()
	}
	return json.Marshal(&alias)
}

func (f *Fragment) UnmarshalJSON(data []byte) error {
	type Alias Fragment
	alias := &struct {
		*Alias
		Error string `json:"error"`
	}{
		Alias: (*Alias)(f),
	}
	err := json.Unmarshal(data, &alias)
	if err != nil {
		return err
	}
	if alias.Error != "" {
		f.Error = errors.New(alias.Error)
	}
	return nil
}

func (r *Resource) MarshalJSON() ([]byte, error) {
	type Alias Resource
	// losing some information here
	errors := make([]string, 0)
	for e := r.Errors.Front(); e != nil; e = e.Next() {
		errors = append(errors, fmt.Sprintf("%v", e.Value))
	}
	fragments := make([]*Fragment, 0, len(r.Fragments))
	for _, f := range r.Fragments {
		fragments = append(fragments, f)
	}
	return json.Marshal(&struct {
		*Alias
		FileMode  uint32      `json:"filemode"`
		Errors    []string    `json:"errors"`
		Fragments []*Fragment `json:"fragments"`
	}{
		Alias:     (*Alias)(r),
		FileMode:  uint32(r.FileMode),
		Errors:    errors,
		Fragments: fragments,
	})
}

func (r *Resource) UnmarshalJSON(data []byte) error {
	type Alias Resource
	alias := &struct {
		FileMode  uint32        `json:"filemode"`
		Errors    []string      `json:"errors"`
		Fragments []*Fragment   `json:"fragments"`
		FragLock  *sync.RWMutex `json:"-"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	r.FileMode = fs.FileMode(alias.FileMode)
	r.Errors = list.New() // may be a need to convert this list to a slice
	for _, e := range alias.Errors {
		r.Errors.PushBack(errors.New(e))
	}
	r.Fragments = make(map[int]*Fragment)
	for _, f := range alias.Fragments {
		r.Fragments[f.Index] = f
	}
	r.FragLock = &sync.RWMutex{}
	return nil
}

//

// Sum of the fragment elapsed times
func (r *Resource) GetElapsedMS() int64 {
	r.FragLock.RLock() // blocks other readers
	defer r.FragLock.RUnlock()

	var elapsedMS int64
	now := time.Now()

	for _, v := range r.Fragments {
		if v.EndTime.IsZero() { // uninitialized
			elapsedMS += now.Sub(v.StartTime).Milliseconds()
		} else {
			elapsedMS += v.EndTime.Sub(v.StartTime).Milliseconds()
		}
	}

	return elapsedMS
}

// Calculated progress percentage as a function of the downloaded bytes and the
// total size. If the total size is unknown, return 0.
func (r *Resource) GetProgess() int64 {
	if r.FileSize == 0 {
		return 0
	}
	var progress int64
	for _, v := range r.Fragments {
		progress += v.Progress
	}
	return int64(float64(progress) / float64(r.FileSize) * 100)
}
