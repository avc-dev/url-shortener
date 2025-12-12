package repository

import (
	"fmt"

	"github.com/avc-dev/url-shortener/internal/model"
)

func (r Repository) CreateURLsBatch(urls map[model.Code]model.URL) error {
	if err := r.underlying.WriteBatch(urls); err != nil {
		return fmt.Errorf("failed to create URLs batch: %w", err)
	}

	return nil
}
