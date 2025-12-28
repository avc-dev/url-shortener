package service

import (
	"fmt"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/model"
)

// URLService содержит бизнес-логику для работы с короткими URL
type URLService struct {
	repo          URLRepository
	codeGenerator Generator
	cfg           *config.Config
}

// NewURLService создает новый экземпляр URLService
func NewURLService(repo URLRepository, cfg *config.Config) *URLService {
	codeGenerator := NewCodeGenerator()
	return &URLService{
		repo:          repo,
		codeGenerator: codeGenerator,
		cfg:           cfg,
	}
}

// CreateShortURL - основная бизнес-логика для создания короткого URL
// Генерирует уникальный код и сохраняет его вместе с оригинальным URL и userID
func (s *URLService) CreateShortURL(originalURL model.URL, userID string) (model.Code, bool, error) {
	// Генерируем уникальный код
	code, err := s.generateUniqueCode()
	if err != nil {
		return "", false, fmt.Errorf("failed to generate unique code: %w", err)
	}

	// Создаем запись или получаем существующую для данного URL и пользователя
	finalCode, created, err := s.repo.CreateOrGetURL(code, originalURL, userID)
	if err != nil {
		return "", false, fmt.Errorf("failed to create or get URL: %w", err)
	}

	return finalCode, created, nil
}

// generateUniqueCode генерирует уникальный код, проверяя его через IsCodeUnique
func (s *URLService) generateUniqueCode() (model.Code, error) {
	for attempt := 0; attempt < s.cfg.Retry.MaxAttempts; attempt++ {
		code := s.codeGenerator.GenerateCode()
		if s.repo.IsCodeUnique(code) {
			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique code after %d attempts: %w", s.cfg.Retry.MaxAttempts, ErrMaxRetriesExceeded)
}

// generateUniqueCodeForBatch генерирует уникальный код для батча, учитывая уже использованные коды в рамках батча
func (s *URLService) generateUniqueCodeForBatch(usedInBatch map[model.Code]bool) (model.Code, error) {
	for attempt := 0; attempt < s.cfg.Retry.MaxAttempts; attempt++ {
		code := s.codeGenerator.GenerateCode()

		// Проверяем конфликт в рамках батча
		if usedInBatch[code] {
			continue
		}

		// Проверяем конфликт в хранилище
		if s.repo.IsCodeUnique(code) {
			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique code for batch after %d attempts: %w", s.cfg.Retry.MaxAttempts, ErrMaxRetriesExceeded)
}

// CreateShortURLsBatch создает короткие URL для нескольких оригинальных URL
// Генерирует уникальные коды для каждого URL и сохраняет их в одной транзакции
func (s *URLService) CreateShortURLsBatch(originalURLs []model.URL, userID string) ([]model.Code, error) {
	urlMap := make(map[model.Code]model.URL)
	usedCodes := make(map[model.Code]bool)

	// Генерируем уникальные коды для каждого URL
	for _, url := range originalURLs {
		code, err := s.generateUniqueCodeForBatch(usedCodes)
		if err != nil {
			return nil, fmt.Errorf("failed to generate unique code for batch: %w", err)
		}
		urlMap[code] = url
		usedCodes[code] = true
	}

	// Сохраняем все URL в одной транзакции
	err := s.repo.CreateURLsBatch(urlMap, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create URLs batch: %w", err)
	}

	// Возвращаем коды в том же порядке, что и входные URL
	codes := make([]model.Code, len(originalURLs))
	for i, url := range originalURLs {
		for code, mappedURL := range urlMap {
			if mappedURL == url {
				codes[i] = code
				break
			}
		}
	}

	return codes, nil
}
