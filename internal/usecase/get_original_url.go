package usecase

import (
	"errors"
	"fmt"

	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/avc-dev/url-shortener/internal/store"
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

		// Проверяем, не удалён ли URL
		if errors.Is(err, store.ErrURLDeleted) {
			return "", fmt.Errorf("%w: %w", ErrURLDeleted, err)
		}

		return "", fmt.Errorf("%w: %w", ErrURLNotFound, err)
	}

	return originalURL.String(), nil
}
