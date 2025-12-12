package service

import (
	"github.com/avc-dev/url-shortener/internal/model"
)

// URLRepository определяет методы для работы с хранилищем URL
type URLRepository interface {
	// CreateURL сохраняет пару код-URL в хранилище
	// Возвращает ErrCodeAlreadyExists если код уже существует или другую ошибку при сохранении
	CreateURL(code model.Code, url model.URL) error
	// CreateURLsBatch сохраняет несколько пар код-URL в хранилище
	// Возвращает ErrCodeAlreadyExists если хотя бы один код уже существует или другую ошибку при сохранении
	CreateURLsBatch(urls map[model.Code]model.URL) error
	// CreateOrGetCode создает новый код для URL или возвращает существующий
	// Возвращает код, признак создания (true если создана новая запись) и ошибку
	CreateOrGetCode(url model.URL) (model.Code, bool, error)
	// GetURLByCode возвращает оригинальный URL по короткому коду
	GetURLByCode(code model.Code) (model.URL, error)
}

// Generator определяет интерфейс для генерации уникальных кодов
type Generator interface {
	// GenerateUniqueCode генерирует уникальный код для заданного URL
	// Возвращает код и ошибку (ErrMaxRetriesExceeded если превышен лимит попыток)
	GenerateUniqueCode(url model.URL) (model.Code, error)
	// GenerateBatchCodes генерирует уникальные коды для батча URL
	// Возвращает мапу кодов на URL и ошибку
	GenerateBatchCodes(urls []model.URL) (map[model.Code]model.URL, error)
}
