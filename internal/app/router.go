package app

import (
	"github.com/avc-dev/url-shortener/internal/handler"
	"github.com/avc-dev/url-shortener/internal/middleware"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// newRouter создает и настраивает роутер приложения
func newRouter(h *handler.Handler, logger *zap.Logger) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger(logger))
	r.Use(middleware.GzipMiddleware(logger))

	// Routes
	r.Post("/", h.CreateURL)
	r.Post("/api/shorten", h.CreateURLJSON)
	r.Get("/{id}", h.GetURL)

	return r
}

