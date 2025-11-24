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

	storage := store.NewStore()
	repo := repository.New(storage)
	urlService := service.NewURLService(repo)
	usecase := handler.New(repo, urlService, cfg)

	r := chi.NewRouter()

	r.Use(middleware.Logger(logger))

	r.Post("/", usecase.CreateURL)
	r.Get("/{id}", usecase.GetURL)

	logger.Info("Starting server", zap.String("address", cfg.ServerAddress.String()))

	err = http.ListenAndServe(cfg.ServerAddress.String(), r)
	if err != nil {
		logger.Fatal("Server failed", zap.Error(err))
	}
}
