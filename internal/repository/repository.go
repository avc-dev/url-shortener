package repository

import "github.com/avc-dev/url-shortener/internal/store"

type Repository struct {
	underlying *store.Store
}

func New(underlying *store.Store) *Repository {
	return &Repository{underlying}
}
