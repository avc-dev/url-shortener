package usecase

import (
	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/model"
	"go.uber.org/zap"
)

// URLRepository определяет интерфейс для работы с хранилищем URL
type URLRepository interface {
	CreateOrGetURL(code model.Code, url model.URL, userID string) (model.Code, bool, error)
	CreateURLsBatch(urls map[model.Code]model.URL, userID string) error
	GetURLByCode(code model.Code) (model.URL, error)
	GetURLsByUserID(userID string, baseURL string) ([]model.UserURLResponse, error)
}

// URLService определяет интерфейс для работы с сервисом генерации коротких URL
type URLService interface {
	CreateShortURL(originalURL model.URL, userID string) (model.Code, bool, error)
	CreateShortURLsBatch(originalURLs []model.URL, userID string) ([]model.Code, error)
}

// URLUsecase содержит бизнес-логику для работы с URL
type URLUsecase struct {
	repo    URLRepository
	service URLService
	cfg     *config.Config
	logger  *zap.Logger
}

// NewURLUsecase создает новый экземпляр URLUsecase
func NewURLUsecase(repo URLRepository, service URLService, cfg *config.Config, logger *zap.Logger) *URLUsecase {
	return &URLUsecase{
		repo:    repo,
		service: service,
		cfg:     cfg,
		logger:  logger,
	}
}
