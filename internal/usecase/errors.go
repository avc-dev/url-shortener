package usecase

import "errors"

var (
	// ErrInvalidURL возвращается, когда переданный URL не прошёл парсинг
	// или не содержит обязательных частей (scheme, host).
	ErrInvalidURL = errors.New("invalid URL")
	// ErrEmptyURL возвращается, когда тело запроса пустое или содержит только пробелы.
	ErrEmptyURL = errors.New("empty URL")
	// ErrServiceUnavailable возвращается при внутренних ошибках (хранилище, генератор кодов).
	ErrServiceUnavailable = errors.New("service unavailable")
	// ErrURLNotFound возвращается, когда короткий код не найден в хранилище.
	ErrURLNotFound = errors.New("URL not found")
	// ErrURLDeleted возвращается, когда URL был найден, но помечен как удалённый.
	ErrURLDeleted = errors.New("URL deleted")
	// ErrURLAlreadyExists — устаревший сентинел; используйте URLAlreadyExistsError для получения кода.
	ErrURLAlreadyExists = errors.New("URL already exists")
)

// URLAlreadyExistsError представляет ошибку дублирования URL с существующим кодом
type URLAlreadyExistsError struct {
	Code string
}

// Error реализует интерфейс error.
func (e URLAlreadyExistsError) Error() string {
	return "URL already exists"
}

// ExistingCode возвращает полный короткий URL уже существующей записи.
func (e URLAlreadyExistsError) ExistingCode() string {
	return e.Code
}
