package main

import (
	"log"
	"net/http"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/handler"
	"github.com/avc-dev/url-shortener/internal/repository"
	"github.com/avc-dev/url-shortener/internal/service"
	"github.com/avc-dev/url-shortener/internal/store"
	"github.com/go-chi/chi/v5"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	storage := store.NewStore()
	repo := repository.New(storage)
	urlService := service.NewURLService(repo)
	usecase := handler.New(repo, urlService, cfg)

	r := chi.NewRouter()
	r.Post("/", usecase.CreateURL)
	r.Get("/{id}", usecase.GetURL)

	log.Printf("Starting server on %s", cfg.ServerAddress.String())

	err = http.ListenAndServe(cfg.ServerAddress.String(), r)
	if err != nil {
		log.Fatal(err)
	}
}
