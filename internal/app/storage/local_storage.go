package storage

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"reflect"

	"github.com/codejago/polypully/downloader/internal/app/model"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// LocalStorage is a storage implementation that uses the local file system
// and is backed by a LevelDB database for persistence. The store is
// intended for use by a single instance of the application and is
// effectively throwaway. It is not intended to be used as a long-term
// storage solution.

var _ LocalStorageApi = (*LocalStorage)(nil)

type LocalStorage struct {
	// configuration for the storage
	config *LocalStorageConfig
	// leveldb options
	options *opt.Options
	// reference to the leveldb database must be closed
	db *leveldb.DB
}

// record is a helper struct for storing records

type record interface {
	Identifier() string
}

func (r *Index) Identifier() string {
	return r.Id
}

// filters resources based on their download status, by convention
type FilterResources func(r *model.Resource) bool

type LocalStorageApi interface {
	// returns a resource from the storage based on a key
	GetResource(id string) (*model.Resource, error)
	// stores a resource in the storage based on a key
	PutResource(id string, value *model.Resource) error
	// returns a list of all resources in the storage
	ListResources(i *Index, filter FilterResources) ([]*model.Resource, error)
	// returns an index from the storage based on an id
	GetIndex(id string) (*Index, error)
	// stores an index in the storage based on an id
	PutIndex(id string, value *Index) error
	// atomically stores a resource and an index in the storage
	PutResourceIndexed(value *model.Resource, index *Index) error
	// closes the storage
	Close()
}

type LocalStorageConfig struct {
	// path to the storage directory
	Path string
	// write buffer size
	BufferMiB int
	// cache for frequently accessed blocks
	CacheMiB int
	// default, snappy, none
	Compression string
	// recovery will be attempted if corruption is detected
	Recovery bool
}

// A new local storage instance backed by leveldb, which is a thread-safe
// key-value store.
func NewLocalStorage(config *LocalStorageConfig) (LocalStorageApi, error) {
	if config == nil {
		return nil, fmt.Errorf("storage config is required")
	}
	if config.Path == "" {
		return nil, fmt.Errorf("storage path is required")
	}
	if fileInfo, err := os.Stat(config.Path); os.IsNotExist(err) || !fileInfo.IsDir() {
		return nil, fmt.Errorf("storage path is not a directory or is inaccessible to the app user")
	}
	options := options(config)
	db, err := leveldb.OpenFile(config.Path, options)
	// if the database is corrupted, go into recovery mode if enabled
	if err != nil {
		switch err.(type) {
		case *errors.ErrCorrupted:
			slog.Warn("storage", "corrupted", err, "attempting recovery", config.Recovery)
			if config.Recovery {
				db, err = leveldb.RecoverFile(config.Path, options)
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("storage error opening db: %v", err)
	}
	return &LocalStorage{config: config, options: options, db: db}, nil
}

// release the os lock on the db files
func (s *LocalStorage) Close() {
	if s.db == nil {
		return
	}
	if err := s.db.Close(); err != nil { // close also flushes the write buffer
		slog.Warn("storage error closing db", "error", err)
	}
}

func options(config *LocalStorageConfig) *opt.Options {
	options := &opt.Options{}
	if config.BufferMiB > 0 {
		options.WriteBuffer = config.BufferMiB * opt.MiB
	} else {
		options.WriteBuffer = 2 * opt.MiB
	}
	if config.CacheMiB > 0 {
		options.BlockCacheCapacity = config.CacheMiB * opt.MiB
	} else {
		options.BlockCacheCapacity = 2 * opt.MiB
	}
	if config.Compression == "snappy" {
		options.Compression = opt.SnappyCompression
	} else if config.Compression == "none" { // less memory, more disk
		options.Compression = opt.NoCompression
	} else {
		options.Compression = opt.DefaultCompression
	}
	return options
}

// Index is a helper struct for indexing resources
// The Index records will probably always be hot
// Avoids iterating over the entire db...
type Index struct {
	// name of the index, unique if used as the key for this record
	Id string `json:"name"`
	// ids of other resources in the storage
	Ids []string `json:"ids"`
}

func (s *LocalStorage) GetIndex(id string) (*Index, error) {
	i, err := get(s, &Index{Id: id})
	if err != nil {
		return nil, err
	}
	if i == nil {
		return &Index{Id: id, Ids: make([]string, 0)}, err
	}
	return *i, err
}

func (s *LocalStorage) PutIndex(id string, value *Index) error {
	return put(s, value)
}

func (s *LocalStorage) GetResource(id string) (*model.Resource, error) {
	r, err := get(s, &model.Resource{Id: id})
	if err != nil {
		return nil, err
	}
	return *r, err
}

func (s *LocalStorage) PutResource(id string, value *model.Resource) error {
	return put(s, value)
}

func (s *LocalStorage) PutResourceIndexed(r *model.Resource, i *Index) error {
	return putb(s, i, r)
}

//

// Keys are the name of the struct and the id of the resource
// and can help to partition the data along the lines of the
// domain model.
func key[U record](value U) []byte {
	name := reflect.TypeOf(value).Elem().Name()
	key := name + "|" + value.Identifier()
	return []byte(key)
}

func get[U record](s *LocalStorage, value U) (*U, error) {
	data, err := s.db.Get(key(value), nil)
	if err != nil && err == errors.ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("storage error getting index: %v", err)
	}
	err = json.Unmarshal(data, value)
	if err != nil {
		return nil, fmt.Errorf("storage error unmarshalling index: %v", err)
	}
	return &value, nil
}

// putb stores multiple records in a batch
// the first record is the index and the rest are resources
func putb[U record, V record](s *LocalStorage, value U, values ...V) error {
	batch := new(leveldb.Batch)
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("storage error marshalling: %v", err)
	}
	batch.Put(key(value), data)
	for _, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("storage error marshalling: %v", err)
		}
		batch.Put(key(value), data)
	}
	if err := s.db.Write(batch, nil); err != nil {
		return fmt.Errorf("storage error storing: %v", err)
	}
	return nil
}

// put stores a single index or resource record
func put[U record](s *LocalStorage, value U) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("storage error marshalling resource: %v", err)
	}
	err = s.db.Put(key(value), data, nil)
	if err != nil {
		return fmt.Errorf("storage error storing resource: %v", err)
	}
	return nil
}

// Returns the resources listed in the index only.
// The filter can be used to further refine the results
func (s *LocalStorage) ListResources(index *Index, filter FilterResources) ([]*model.Resource, error) {
	if index == nil {
		return nil, fmt.Errorf("storage index is required")
	}
	resources := make([]*model.Resource, 0)
	if len(index.Ids) == 0 {
		return resources, nil
	}
	value := &model.Resource{}
	for _, id := range index.Ids {
		value.Id = id
		data, err := s.db.Get(key(value), nil)
		if err != nil {
			return nil, fmt.Errorf("storage error getting index: %v", err)
		}
		resource := model.Resource{}
		err = json.Unmarshal(data, &resource)
		if err != nil {
			return nil, fmt.Errorf("storage error unmarshalling resource: %v", err)
		}
		if filter == nil || filter(&resource) {
			resources = append(resources, &resource)
		}
	}
	return resources, nil
}
