package repository

import (
	"github.com/avc-dev/url-shortener/internal/model"
)

type Store interface {
	Read(key model.Code) (model.URL, error)
	Write(key model.Code, value model.URL) error
	WriteBatch(urls map[model.Code]model.URL) error
}

type Repository struct {
	underlying Store
}

func New(underlying Store) *Repository {
	return &Repository{underlying}
}
