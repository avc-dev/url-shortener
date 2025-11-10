package service

import (
	"fmt"
	"math/rand"

	"github.com/avc-dev/url-shortener/internal/model"
)

const (
	CodeLength   = 8
	MaxTries     = 100
	AllowedChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// URLService содержит бизнес-логику для работы с короткими URL
type URLService struct {
	checker CodeChecker
}

// NewURLService создает новый экземпляр URLService
func NewURLService(checker CodeChecker) *URLService {
	return &URLService{checker: checker}
}

// randomString генерирует случайную строку заданной длины
func randomString() string {
	result := make([]byte, CodeLength)

	for i := range result {
		result[i] = AllowedChars[rand.Intn(len(AllowedChars))]
	}

	return string(result)
}

// GenerateUniqueCode генерирует уникальный код, проверяя его существование в хранилище
// Возвращает ошибку если не удалось сгенерировать уникальный код за MaxTries попыток
func (s *URLService) GenerateUniqueCode() (model.Code, error) {
	for tries := 0; tries < MaxTries; tries++ {
		code := model.Code(randomString())

		exists, err := s.checker.Exists(code)
		if err != nil {
			// Логируем ошибку, но продолжаем попытки
			continue
		}

		if !exists {
			return code, nil
		}
	}

	return "", fmt.Errorf("could not generate unique code after %d tries", MaxTries)
}

// CreateShortURL - основная бизнес-логика для создания короткого URL
// Генерирует уникальный код для переданного оригинального URL
func (s *URLService) CreateShortURL(originalURL model.URL) (model.Code, error) {
	code, err := s.GenerateUniqueCode()
	if err != nil {
		return "", fmt.Errorf("failed to generate unique code: %w", err)
	}

	return code, nil
}
