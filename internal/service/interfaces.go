package service

import "github.com/avc-dev/url-shortener/internal/model"

// URLRepository определяет методы для работы с хранилищем URL
type URLRepository interface {
	// CreateURL сохраняет пару код-URL в хранилище
	// Возвращает ошибку если код уже существует или произошла ошибка при сохранении
	CreateURL(code model.Code, url model.URL) error
}
