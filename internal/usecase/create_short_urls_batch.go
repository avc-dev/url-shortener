package usecase

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/avc-dev/url-shortener/internal/model"
	"go.uber.org/zap"
)

// CreateShortURLsBatch создает короткие URL для нескольких строковых URL
// Выполняет валидацию, очистку URL и генерацию коротких кодов для каждого
func (u *URLUsecase) CreateShortURLsBatch(urlStrings []string, userID string) ([]string, error) {
	originalURLs := make([]model.URL, len(urlStrings))

	// Валидируем и очищаем все URL
	for i, urlString := range urlStrings {
		urlString = strings.TrimSpace(urlString)
		urlString = strings.Trim(urlString, `"'`)

		if urlString == "" {
			return nil, fmt.Errorf("%w: empty URL at index %d", ErrEmptyURL, i)
		}

		parsedURL, err := url.Parse(urlString)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid URL at index %d: %w", ErrInvalidURL, i, err)
		}

		if parsedURL.Scheme == "" {
			return nil, fmt.Errorf("%w: scheme is missing at index %d", ErrInvalidURL, i)
		}

		if parsedURL.Host == "" {
			return nil, fmt.Errorf("%w: host is missing at index %d", ErrInvalidURL, i)
		}

		originalURLs[i] = model.URL(urlString)
	}

	// Создаем короткие URL через сервис
	codes, err := u.service.CreateShortURLsBatch(originalURLs, userID)
	if err != nil {
		u.logger.Error("failed to create short URLs batch",
			zap.Strings("original_urls", urlStrings),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("%w: %w", ErrServiceUnavailable, err)
	}

	// Формируем полные короткие URL
	shortURLs := make([]string, len(codes))
	for i, code := range codes {
		shortURL, err := url.JoinPath(u.cfg.BaseURL.String(), string(code))
		if err != nil {
			u.logger.Error("failed to build short URL",
				zap.String("base_url", u.cfg.BaseURL.String()),
				zap.String("code", string(code)),
				zap.Error(err),
			)
			return nil, fmt.Errorf("%w: failed to build short URL: %w", ErrServiceUnavailable, err)
		}
		shortURLs[i] = shortURL
	}

	return shortURLs, nil
}
