package repository

import (
	"fmt"

	"github.com/avc-dev/url-shortener/internal/model"
)

func (r Repository) CreateURL(code model.Code, url model.URL) error {
	if err := r.underlying.Write(code, url); err != nil {
		return fmt.Errorf("failed to create URL: %w", err)
	}

	return nil
}
