package service

import (
	"fmt"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/avc-dev/url-shortener/internal/store"
)

// URLService содержит бизнес-логику для работы с короткими URL
type URLService struct {
	repo          URLRepository
	codeGenerator Generator
}

// NewURLService создает новый экземпляр URLService
func NewURLService(repo URLRepository, cfg *config.Config) *URLService {
	codeGenerator := NewCodeGenerator(repo, cfg)
	return &URLService{
		repo:          repo,
		codeGenerator: codeGenerator,
	}
}

// CreateShortURL - основная бизнес-логика для создания короткого URL
// Генерирует уникальный код и сохраняет его вместе с оригинальным URL
// Если URL уже существует, возвращает ошибку с существующим кодом
func (s *URLService) CreateShortURL(originalURL model.URL) (model.Code, error) {
	// Сначала проверяем, существует ли уже такой URL
	code, created, err := s.repo.CreateOrGetCode(originalURL)
	if err != nil {
		return "", fmt.Errorf("failed to check URL existence: %w", err)
	}

	// Если URL уже существовал, возвращаем ошибку
	if !created {
		return code, fmt.Errorf("URL already exists: %w", store.ErrURLAlreadyExists)
	}

	// Генерируем уникальный код и сохраняем
	code, err = s.codeGenerator.GenerateUniqueCode(originalURL)
	if err != nil {
		return "", fmt.Errorf("failed to generate unique code: %w", err)
	}

	// Сохраняем код-URL пару в хранилище
	err = s.repo.CreateURL(code, originalURL)
	if err != nil {
		return "", fmt.Errorf("failed to save URL: %w", err)
	}

	return code, nil
}

// CreateShortURLsBatch создает короткие URL для нескольких оригинальных URL
// Генерирует уникальные коды для каждого URL и сохраняет их в одной транзакции
func (s *URLService) CreateShortURLsBatch(originalURLs []model.URL) ([]model.Code, error) {
	// Генерируем уникальные коды для всех URL
	urlMap, err := s.codeGenerator.GenerateBatchCodes(originalURLs)
	if err != nil {
		return nil, fmt.Errorf("failed to generate unique codes: %w", err)
	}

	// Сохраняем все URL в одной транзакции
	err = s.repo.CreateURLsBatch(urlMap)
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
