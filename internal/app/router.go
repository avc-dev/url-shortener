package app

import (
	"github.com/avc-dev/url-shortener/internal/handler"
	"github.com/avc-dev/url-shortener/internal/middleware"
	"github.com/avc-dev/url-shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// newRouter создает и настраивает роутер приложения
func newRouter(h *handler.Handler, logger *zap.Logger, authService *service.AuthService) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger(logger))
	r.Use(middleware.GzipMiddleware(logger))

	// Auth
	authMiddleware := middleware.NewAuthMiddleware(authService, logger)

	// Routes
	r.Get("/ping", h.Ping)
	r.Get("/{id}", h.GetURL)

	// Authenticated routes - маршруты создания URL с опциональной аутентификацией
	r.With(authMiddleware.OptionalAuth).Post("/", h.CreateURL)
	r.With(authMiddleware.OptionalAuth).Post("/api/shorten", h.CreateURLJSON)
	r.With(authMiddleware.OptionalAuth).Post("/api/shorten/batch", h.CreateURLBatch)

	// User URLs routes - требуют аутентификации
	r.With(authMiddleware.RequireAuth).Get("/api/user/urls", h.GetUserURLs)
	r.With(authMiddleware.RequireAuth).Delete("/api/user/urls", h.DeleteURLs)

	return r
}
