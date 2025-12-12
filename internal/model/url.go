package model

type Code string

type URL string

func (U URL) String() string {
	return string(U)
}

// URLEntry представляет запись URL с уникальным идентификатором для хранения
type URLEntry struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
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
