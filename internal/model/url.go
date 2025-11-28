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
