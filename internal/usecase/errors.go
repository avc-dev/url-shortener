package usecase

import "errors"

var (
	ErrInvalidURL         = errors.New("invalid URL")
	ErrEmptyURL           = errors.New("empty URL")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrURLNotFound        = errors.New("URL not found")
	ErrURLDeleted         = errors.New("URL deleted")
	ErrURLAlreadyExists   = errors.New("URL already exists")
)

// URLAlreadyExistsError представляет ошибку дублирования URL с существующим кодом
type URLAlreadyExistsError struct {
	Code string
}

func (e URLAlreadyExistsError) Error() string {
	return "URL already exists"
}

func (e URLAlreadyExistsError) ExistingCode() string {
	return e.Code
}


