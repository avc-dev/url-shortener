package repository

import "github.com/avc-dev/url-shortener/internal/model"

func (r Repository) GetURLByCode(code model.Code) (model.URL, error) {
	url, err := r.underlying.Read(code)

	if err != nil {
		return "", err
	}

	return url, nil
}
