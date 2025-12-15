package service

import (
	"github.com/avc-dev/url-shortener/internal/model"
)

// URLRepository определяет методы для работы с хранилищем URL
type URLRepository interface {
	// CreateURL сохраняет пару код-URL в хранилище
	// Возвращает ErrCodeAlreadyExists если код уже существует или другую ошибку при сохранении
	CreateURL(code model.Code, url model.URL) error
	// CreateOrGetURL создает новую запись или возвращает код существующей для данного URL
	// Возвращает код и признак создания (true если создана новая запись)
	CreateOrGetURL(code model.Code, url model.URL) (model.Code, bool, error)
	// CreateURLsBatch сохраняет несколько пар код-URL в хранилище
	// Возвращает ErrCodeAlreadyExists если хотя бы один код уже существует или другую ошибку при сохранении
	CreateURLsBatch(urls map[model.Code]model.URL) error
	// GetURLByCode возвращает оригинальный URL по короткому коду
	GetURLByCode(code model.Code) (model.URL, error)
	// IsCodeUnique проверяет, свободен ли код
	IsCodeUnique(code model.Code) bool
	// GetCodeByURL возвращает код для существующего URL или ошибку если URL не найден
	GetCodeByURL(url model.URL) (model.Code, error)
}

// Generator определяет интерфейс для генерации кодов
type Generator interface {
	// GenerateCode генерирует случайный код
	GenerateCode() model.Code
	// GenerateBatchCodes генерирует указанное количество случайных кодов
	GenerateBatchCodes(count int) []model.Code
}
