package usecase

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/avc-dev/url-shortener/internal/model"
	"go.uber.org/zap"
)

// CreateShortURLFromString создает короткий URL из строки оригинального URL
// Выполняет валидацию, очистку URL и генерацию короткого кода
func (u *URLUsecase) CreateShortURLFromString(urlString string) (string, error) {
	urlString = strings.TrimSpace(urlString)
	urlString = strings.Trim(urlString, `"'`)

	if urlString == "" {
		return "", ErrEmptyURL
	}

	parsedURL, err := url.Parse(urlString)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", ErrInvalidURL
	}

	originalURL := model.URL(urlString)
	code, err := u.service.CreateShortURL(originalURL)
	if err != nil {
		u.logger.Error("failed to create short URL",
			zap.String("original_url", string(originalURL)),
			zap.Error(err),
		)
		return "", fmt.Errorf("%w: %w", ErrServiceUnavailable, err)
	}

	shortURL, err := url.JoinPath(u.cfg.BaseURL.String(), string(code))
	if err != nil {
		u.logger.Error("failed to build short URL",
			zap.String("base_url", u.cfg.BaseURL.String()),
			zap.String("code", string(code)),
			zap.Error(err),
		)
		return "", fmt.Errorf("%w: failed to build short URL: %w", ErrServiceUnavailable, err)
	}

	return shortURL, nil
}
