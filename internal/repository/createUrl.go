package repository

import "github.com/avc-dev/url-shortener/internal/model"

func (r Repository) CreateURL(code model.Code, url model.URL) error {
	if err := r.underlying.Write(code, url); err != nil {
		return err
	}

	return nil
}
