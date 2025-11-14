package repository

import (
	"fmt"

	"github.com/avc-dev/url-shortener/internal/model"
)

func (r Repository) GetURLByCode(code model.Code) (model.URL, error) {
	url, err := r.underlying.Read(code)

	if err != nil {
		return "", fmt.Errorf("failed to get URL by code: %w", err)
	}

	return url, nil
}
