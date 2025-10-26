package main

import (
	"net/http"

	"github.com/avc-dev/url-shortener/internal/handler"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, handler.CreateUrl)
	mux.HandleFunc(`/EwHXdJfB`, handler.GetUrl)

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
