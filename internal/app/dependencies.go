package app

import (
	"fmt"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/handler"
	"github.com/avc-dev/url-shortener/internal/repository"
	"github.com/avc-dev/url-shortener/internal/service"
	"github.com/avc-dev/url-shortener/internal/store"
	"github.com/avc-dev/url-shortener/internal/usecase"
	"go.uber.org/zap"
)

// initDependencies инициализирует все зависимости приложения
func initDependencies(cfg *config.Config, logger *zap.Logger) (*handler.Handler, error) {
	storage, err := initStorage(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	repo := repository.New(storage)
	urlService := service.NewURLService(repo)
	urlUsecase := usecase.NewURLUsecase(repo, urlService, cfg, logger)
	h := handler.New(urlUsecase, logger)

	return h, nil
}

// initStorage создает хранилище на основе конфигурации
func initStorage(cfg *config.Config, logger *zap.Logger) (repository.Store, error) {
	if cfg.FileStoragePath != "" {
		fileStore, err := store.NewFileStore(cfg.FileStoragePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create file store: %w", err)
		}
		logger.Info("Using file storage", zap.String("path", cfg.FileStoragePath))
		return fileStore, nil
	}

	logger.Info("Using in-memory storage")
	return store.NewStore(), nil
}
