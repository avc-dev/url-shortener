package main

import (
	"flag"
	"net/http"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/handler"
	"github.com/avc-dev/url-shortener/internal/repository"
	"github.com/avc-dev/url-shortener/internal/store"
	"github.com/go-chi/chi/v5"
)

func main() {
	flag.Parse()

	storage := store.NewStore()
	repo := repository.New(storage)
	usecase := handler.New(repo)

	r := chi.NewRouter()
	r.Post("/", usecase.CreateURL)
	r.Get("/{id}", usecase.GetURL)

	err := http.ListenAndServe(config.Address.String(), r)
	if err != nil {
		panic(err)
	}
}
