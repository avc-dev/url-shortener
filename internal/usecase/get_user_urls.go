package usecase

import (
	"fmt"

	"github.com/avc-dev/url-shortener/internal/model"
	"go.uber.org/zap"
)

// GetURLsByUserID возвращает все URL для указанного пользователя
func (u *URLUsecase) GetURLsByUserID(userID string) ([]model.UserURLResponse, error) {
	u.logger.Info("GetURLsByUserID called", zap.String("user_id", userID))

	urls, err := u.repo.GetURLsByUserID(userID, u.cfg.BaseURL.String())
	if err != nil {
		u.logger.Error("failed to get URLs by user ID",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("%w: %w", ErrServiceUnavailable, err)
	}

	u.logger.Info("GetURLsByUserID result", zap.Int("urls_count", len(urls)))
	return urls, nil
}
