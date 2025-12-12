package repository

import (
	"fmt"

	"github.com/avc-dev/url-shortener/internal/model"
)

type Store interface {
	Read(key model.Code) (model.URL, error)
	Write(key model.Code, value model.URL) error
	WriteBatch(urls map[model.Code]model.URL) error
	CreateOrGetCode(value model.URL) (model.Code, bool, error)
}

type Repository struct {
	underlying Store
}

func New(underlying Store) *Repository {
	return &Repository{underlying}
}

func (r Repository) CreateOrGetCode(url model.URL) (model.Code, bool, error) {
	code, created, err := r.underlying.CreateOrGetCode(url)
	if err != nil {
		return "", false, fmt.Errorf("failed to create or get code: %w", err)
	}

	return code, created, nil
}
