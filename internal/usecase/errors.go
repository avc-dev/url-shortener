package usecase

import "errors"

var (
	ErrInvalidURL         = errors.New("invalid URL")
	ErrEmptyURL           = errors.New("empty URL")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrURLNotFound        = errors.New("URL not found")
)

