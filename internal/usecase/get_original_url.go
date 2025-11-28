package usecase

import (
	"fmt"

	"github.com/avc-dev/url-shortener/internal/model"
	"go.uber.org/zap"
)

// GetOriginalURL получает оригинальный URL по короткому коду
func (u *URLUsecase) GetOriginalURL(code string) (string, error) {
	originalURL, err := u.repo.GetURLByCode(model.Code(code))
	if err != nil {
		u.logger.Error("failed to get URL by code",
			zap.String("code", code),
			zap.Error(err),
		)
		return "", fmt.Errorf("%w: %w", ErrURLNotFound, err)
	}

	return originalURL.String(), nil
}
