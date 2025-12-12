package service

import (
	"fmt"
	"math/rand"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/model"
)

const (
	CodeLength   = 8
	AllowedChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// CodeGenerator реализует CodeGenerator с использованием вероятностного подхода
type CodeGenerator struct {
	repo        URLRepository
	maxAttempts int
	random      *rand.Rand
}

// NewCodeGenerator создает новый генератор кодов
func NewCodeGenerator(repo URLRepository, cfg *config.Config) *CodeGenerator {
	return &CodeGenerator{
		repo:        repo,
		maxAttempts: cfg.Retry.MaxAttempts,
		random:      rand.New(rand.NewSource(rand.Int63())),
	}
}

// GenerateUniqueCode генерирует уникальный код для заданного URL
func (g *CodeGenerator) GenerateUniqueCode(url model.URL) (model.Code, error) {
	for attempt := 0; attempt < g.maxAttempts; attempt++ {
		code := model.Code(g.generateRandomString())

		// Проверяем, существует ли такой код
		_, err := g.repo.GetURLByCode(code)
		if err != nil {
			// Если код не найден, значит он свободен
			return code, nil
		}
		// Если код найден, продолжаем генерацию
	}

	return "", fmt.Errorf("%w: failed to generate unique code after %d attempts", ErrMaxRetriesExceeded, g.maxAttempts)
}

// GenerateBatchCodes генерирует уникальные коды для батча URL
func (g *CodeGenerator) GenerateBatchCodes(urls []model.URL) (map[model.Code]model.URL, error) {
	result := make(map[model.Code]model.URL)
	usedCodes := make(map[model.Code]bool)

	for _, url := range urls {
		code, err := g.generateUniqueCodeForBatch(usedCodes)
		if err != nil {
			return nil, err
		}
		result[code] = url
		usedCodes[code] = true
	}

	return result, nil
}

// generateUniqueCodeForBatch генерирует уникальный код для батча, учитывая уже использованные коды в рамках батча
func (g *CodeGenerator) generateUniqueCodeForBatch(usedInBatch map[model.Code]bool) (model.Code, error) {
	for attempt := 0; attempt < g.maxAttempts; attempt++ {
		code := model.Code(g.generateRandomString())

		// Проверяем конфликт в рамках батча
		if usedInBatch[code] {
			continue
		}

		// Проверяем конфликт в хранилище
		_, err := g.repo.GetURLByCode(code)
		if err != nil {
			// Код свободен
			return code, nil
		}
		// Код занят в хранилище, продолжаем
	}

	return "", fmt.Errorf("%w: failed to generate unique code for batch after %d attempts", ErrMaxRetriesExceeded, g.maxAttempts)
}

// generateRandomString генерирует случайную строку заданной длины
func (g *CodeGenerator) generateRandomString() string {
	result := make([]byte, CodeLength)

	for i := range result {
		result[i] = AllowedChars[g.random.Intn(len(AllowedChars))]
	}

	return string(result)
}
