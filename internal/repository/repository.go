package repository

import (
	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/avc-dev/url-shortener/internal/store"
)

type Store interface {
	Read(key model.Code) (model.URL, error)
	Write(key model.Code, value model.URL) error
}

type Repository struct {
	underlying Store
}

func New(underlying Store) *Repository {
	return &Repository{underlying}
}
