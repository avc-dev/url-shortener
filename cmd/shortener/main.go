package main

import (
	"net/http"

	"github.com/avc-dev/url-shortener/internal/handler"
	"github.com/avc-dev/url-shortener/internal/repository"
	"github.com/avc-dev/url-shortener/internal/store"
	"github.com/go-chi/chi/v5"
)

func main() {
	storage := store.NewStore()
	repo := repository.New(storage)
	usecase := handler.New(repo)

	r := chi.NewRouter()
	r.Post("/", usecase.CreateURL)
	r.Get("/{id}", usecase.GetURL)

	// TODO cfg: move to config
	err := http.ListenAndServe(`:8080`, r)
	if err != nil {
		panic(err)
	}
}
