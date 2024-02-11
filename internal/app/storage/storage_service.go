package storage

import (
	"github.com/codejago/polypully/downloader/internal/app/model"
)

var _ StorageApi = (*Storage)(nil)

const (
	DownloadsIndex = "downloads"
)

type Storage struct {
	localStorage LocalStorageApi
}

type StorageApi interface {
	UpdateResource(value *model.Resource) error
	GetResource(id string) (*model.Resource, *Index, error)
	ListResources(filter FilterResources) ([]*model.Resource, error)
}

func NewStorage(localStorage LocalStorageApi) StorageApi {
	return &Storage{localStorage: localStorage}
}

func (s *Storage) ListResources(filter FilterResources) ([]*model.Resource, error) {
	index, err := s.localStorage.GetIndex(DownloadsIndex)
	if err != nil {
		return nil, err
	}
	return s.localStorage.ListResources(index, filter)
}

func (s *Storage) GetResource(id string) (*model.Resource, *Index, error) {
	r, err := s.localStorage.GetResource(id)
	if err != nil {
		return nil, nil, err
	}
	i, err := s.localStorage.GetIndex(DownloadsIndex)
	if err != nil {
		return nil, nil, err
	}
	return r, i, nil
}

func (s *Storage) UpdateResource(value *model.Resource) error {
	index, err := s.localStorage.GetIndex(DownloadsIndex)
	if err != nil {
		return err
	}
	ids := make([]string, 0)
	for _, id := range index.Ids {
		if id != value.Id {
			ids = append(ids, id)
		}
	}
	// if value.Status != model.DownloadComplete {
	ids = append(ids, value.Id)
	// }
	index.Ids = ids
	return s.localStorage.PutResourceIndexed(value, index)
}
