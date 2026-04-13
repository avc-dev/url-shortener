// Package model определяет доменные типы и структуры данных URL-сокращателя.
package model

// Code — тип короткого кода, идентифицирующего оригинальный URL в хранилище.
type Code string

// URL — тип оригинального URL, для которого создаётся короткая ссылка.
type URL string

// String возвращает строковое представление URL.
func (U URL) String() string {
	return string(U)
}

// URLEntry представляет запись URL с уникальным идентификатором для хранения
type URLEntry struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id,omitempty"`
	DeletedFlag bool   `json:"is_deleted,omitempty"`
}

// BatchShortenRequest представляет элемент запроса для батчевого сокращения URL
type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// BatchShortenResponse представляет элемент ответа для батчевого сокращения URL
type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// UserURLResponse представляет элемент ответа для получения URL пользователя
type UserURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// Stats содержит агрегированную статистику сервиса.
type Stats struct {
	URLCount  int
	UserCount int
}
