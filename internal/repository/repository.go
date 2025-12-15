package repository

import (
	"fmt"

	"github.com/avc-dev/url-shortener/internal/model"
)

type Store interface {
	Read(key model.Code) (model.URL, error)
	Write(key model.Code, value model.URL) error
	WriteBatch(urls map[model.Code]model.URL) error
	CreateOrGetURL(code model.Code, url model.URL) (model.Code, bool, error)
	IsCodeUnique(code model.Code) bool
	GetCodeByURL(url model.URL) (model.Code, error)
}

type Repository struct {
	underlying Store
}

func New(underlying Store) *Repository {
	return &Repository{underlying}
}

func (r Repository) IsCodeUnique(code model.Code) bool {
	return r.underlying.IsCodeUnique(code)
}

func (r Repository) GetCodeByURL(url model.URL) (model.Code, error) {
	code, err := r.underlying.GetCodeByURL(url)
	if err != nil {
		return "", fmt.Errorf("failed to get code by URL: %w", err)
	}

	return code, nil
}

func (r Repository) CreateOrGetURL(code model.Code, url model.URL) (model.Code, bool, error) {
	finalCode, created, err := r.underlying.CreateOrGetURL(code, url)
	if err != nil {
		return "", false, fmt.Errorf("failed to create or get URL: %w", err)
	}

	return finalCode, created, nil
}
