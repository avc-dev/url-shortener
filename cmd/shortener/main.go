package main

import (
	"log"
	"net/http"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/handler"
	"github.com/avc-dev/url-shortener/internal/middleware"
	"github.com/avc-dev/url-shortener/internal/repository"
	"github.com/avc-dev/url-shortener/internal/service"
	"github.com/avc-dev/url-shortener/internal/store"
	"github.com/avc-dev/url-shortener/internal/usecase"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	// Создаём хранилище в зависимости от конфигурации
	var storage repository.Store
	if cfg.FileStoragePath != "" {
		// Используем FileStore для персистентного хранения
		fileStore, err := store.NewFileStore(cfg.FileStoragePath)
		if err != nil {
			logger.Fatal("Failed to initialize file store", zap.Error(err))
		}
		storage = fileStore
		logger.Info("Using file storage", zap.String("path", cfg.FileStoragePath))
	} else {
		// Используем обычный in-memory Store
		storage = store.NewStore()
		logger.Info("Using in-memory storage")
	}

	repo := repository.New(storage)
	urlService := service.NewURLService(repo)
	urlUsecase := usecase.NewURLUsecase(repo, urlService, cfg, logger)
	h := handler.New(urlUsecase, logger)

	r := chi.NewRouter()

	r.Use(middleware.Logger(logger))
	r.Use(middleware.GzipMiddleware(logger))

	r.Post("/", h.CreateURL)
	r.Post("/api/shorten", h.CreateURLJSON)
	r.Get("/{id}", h.GetURL)

	logger.Info("Starting server", zap.String("address", cfg.ServerAddress.String()))

	err = http.ListenAndServe(cfg.ServerAddress.String(), r)
	if err != nil {
		logger.Fatal("Server failed", zap.Error(err))
	}
}
