package repository

import (
	"fmt"

	"github.com/avc-dev/url-shortener/internal/model"
)

type Store interface {
	Read(key model.Code) (model.URL, error)
	Write(key model.Code, value model.URL, userID string) error
	WriteBatch(urls map[model.Code]model.URL, userID string) error
	CreateOrGetURL(code model.Code, url model.URL, userID string) (model.Code, bool, error)
	IsCodeUnique(code model.Code) bool
	GetURLsByUserID(userID string, baseURL string) ([]model.UserURLResponse, error)
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

func (r Repository) Write(code model.Code, url model.URL, userID string) error {
	err := r.underlying.Write(code, url, userID)
	if err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}
	return nil
}

func (r Repository) CreateOrGetURL(code model.Code, url model.URL, userID string) (model.Code, bool, error) {
	finalCode, created, err := r.underlying.CreateOrGetURL(code, url, userID)
	if err != nil {
		return "", false, fmt.Errorf("failed to create or get URL: %w", err)
	}

	return finalCode, created, nil
}

func (r Repository) CreateURLsBatch(urls map[model.Code]model.URL, userID string) error {
	err := r.underlying.WriteBatch(urls, userID)
	if err != nil {
		return fmt.Errorf("failed to create URLs batch: %w", err)
	}
	return nil
}

func (r Repository) GetURLsByUserID(userID string, baseURL string) ([]model.UserURLResponse, error) {
	urls, err := r.underlying.GetURLsByUserID(userID, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get URLs by user ID: %w", err)
	}
	return urls, nil
}
