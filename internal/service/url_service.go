package service

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/avc-dev/url-shortener/internal/store"
)

const (
	CodeLength   = 8
	MaxTries     = 100
	AllowedChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// URLService содержит бизнес-логику для работы с короткими URL
type URLService struct {
	repo URLRepository
}

// NewURLService создает новый экземпляр URLService
func NewURLService(repo URLRepository) *URLService {
	return &URLService{repo: repo}
}

// randomString генерирует случайную строку заданной длины
func randomString() string {
	result := make([]byte, CodeLength)

	for i := range result {
		result[i] = AllowedChars[rand.Intn(len(AllowedChars))]
	}

	return string(result)
}

// CreateShortURL - основная бизнес-логика для создания короткого URL
// Генерирует уникальный код и сохраняет его вместе с оригинальным URL
// Использует retry механизм для обработки коллизий кодов
func (s *URLService) CreateShortURL(originalURL model.URL) (model.Code, error) {
	for tries := 0; tries < MaxTries; tries++ {
		code := model.Code(randomString())

		err := s.repo.CreateURL(code, originalURL)
		if err != nil {
			// Если коллизия - пробуем еще раз
			if errors.Is(err, store.ErrAlreadyExists) {
				continue
			}
			// Любая другая ошибка - возвращаем сразу
			return "", fmt.Errorf("failed to create URL: %w", err)
		}

		// Успешно сохранили
		return code, nil
	}

	return "", fmt.Errorf("%w: after %d tries", ErrMaxRetriesExceeded, MaxTries)
}
