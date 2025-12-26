package service

import (
	"github.com/avc-dev/url-shortener/internal/model"
)

// URLRepository определяет методы для работы с хранилищем URL
type URLRepository interface {
	// CreateOrGetURL создает новую запись или возвращает код существующей для данного URL и пользователя
	CreateOrGetURL(code model.Code, url model.URL, userID string) (model.Code, bool, error)
	// CreateURLsBatch сохраняет несколько пар код-URL для пользователя
	CreateURLsBatch(urls map[model.Code]model.URL, userID string) error
	// GetURLByCode возвращает оригинальный URL по короткому коду
	GetURLByCode(code model.Code) (model.URL, error)
	// GetURLsByUserID возвращает все URL для указанного пользователя
	GetURLsByUserID(userID string, baseURL string) ([]model.UserURLResponse, error)
	// IsCodeUnique проверяет, свободен ли код
	IsCodeUnique(code model.Code) bool
}

// Generator определяет интерфейс для генерации кодов
type Generator interface {
	// GenerateCode генерирует случайный код
	GenerateCode() model.Code
	// GenerateBatchCodes генерирует указанное количество случайных кодов
	GenerateBatchCodes(count int) []model.Code
}
