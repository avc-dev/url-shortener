package app

import (
	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/handler"
	"github.com/avc-dev/url-shortener/internal/middleware"
	"github.com/avc-dev/url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// newRouter создает и настраивает роутер приложения
func newRouter(h *handler.Handler, logger *zap.Logger, cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger(logger))
	r.Use(middleware.GzipMiddleware(logger))

	// Auth
	authService := service.NewAuthService(cfg.JWTSecret)
	authMiddleware := middleware.NewAuthMiddleware(authService, logger)

	// Routes
	r.Get("/ping", h.Ping)
	r.Get("/{id}", h.GetURL)

	// Authenticated routes - все маршруты создания URL требуют аутентификации
	r.With(authMiddleware.Authenticate).Post("/", h.CreateURL)
	r.With(authMiddleware.Authenticate).Post("/api/shorten", h.CreateURLJSON)
	r.With(authMiddleware.Authenticate).Post("/api/shorten/batch", h.CreateURLBatch)

	// User URLs route - требует строгой аутентификации
	r.With(authMiddleware.RequireAuth).Get("/api/user/urls", h.GetUserURLs)

	return r
}
