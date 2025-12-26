package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/avc-dev/url-shortener/internal/config/db"
	"github.com/avc-dev/url-shortener/internal/middleware"
	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/avc-dev/url-shortener/internal/service"
	"github.com/avc-dev/url-shortener/internal/usecase"
	"go.uber.org/zap"
)

// URLUsecase определяет интерфейс для бизнес-логики работы с URL
type URLUsecase interface {
	CreateShortURLFromString(urlString string, userID string) (string, error)
	CreateShortURLsBatch(urlStrings []string, userID string) ([]string, error)
	GetOriginalURL(code string) (string, error)
	GetURLsByUserID(userID string) ([]model.UserURLResponse, error)
}

// Handler обрабатывает HTTP запросы
type Handler struct {
	usecase     URLUsecase
	logger      *zap.Logger
	dbPool      db.Database
	authService *service.AuthService
}

// New создает новый экземпляр Handler
func New(usecase URLUsecase, logger *zap.Logger, dbPool db.Database, authService *service.AuthService) *Handler {
	return &Handler{
		usecase:     usecase,
		logger:      logger,
		dbPool:      dbPool,
		authService: authService,
	}
}

// Ping проверяет подключение к базе данных
func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	if h.dbPool == nil {
		h.logger.Error("database not configured")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	if err := db.Ping(ctx, h.dbPool); err != nil {
		h.logger.Error("database ping failed", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleError маппит ошибки usecase на HTTP статусы
func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidURL), errors.Is(err, usecase.ErrEmptyURL):
		h.logger.Debug("bad request", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
	case errors.Is(err, usecase.ErrURLNotFound):
		h.logger.Debug("URL not found", zap.Error(err))
		w.WriteHeader(http.StatusNotFound)
	case errors.Is(err, usecase.ErrURLAlreadyExists):
		h.logger.Debug("URL already exists", zap.Error(err))
		w.WriteHeader(http.StatusConflict)
	default:
		var urlExistsErr usecase.URLAlreadyExistsError
		if errors.As(err, &urlExistsErr) {
			h.logger.Debug("URL already exists", zap.Error(err))
			w.WriteHeader(http.StatusConflict)
			return
		}
		h.logger.Error("internal server error", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// handleErrorJSON обрабатывает ошибки для JSON API endpoints
func (h *Handler) handleErrorJSON(w http.ResponseWriter, err error) {
	var urlExistsErr usecase.URLAlreadyExistsError
	if errors.As(err, &urlExistsErr) {
		// Для JSON API при дублировании URL возвращаем существующий код
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		response := map[string]string{"result": urlExistsErr.ExistingCode()}
		if jsonErr := json.NewEncoder(w).Encode(response); jsonErr != nil {
			h.logger.Error("failed to encode JSON response", zap.Error(jsonErr))
		}
		return
	}

	// Для остальных ошибок используем обычную обработку
	h.handleError(w, err)
}

// getUserIDFromRequest извлекает user_id из контекста запроса
func (h *Handler) getUserIDFromRequest(r *http.Request) (string, bool) {
	return middleware.GetUserIDFromContext(r.Context())
}
