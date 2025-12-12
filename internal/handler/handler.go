package handler

import (
	"errors"
	"net/http"

	"github.com/avc-dev/url-shortener/internal/config/db"
	"github.com/avc-dev/url-shortener/internal/usecase"
	"go.uber.org/zap"
)

// URLUsecase определяет интерфейс для бизнес-логики работы с URL
type URLUsecase interface {
	CreateShortURLFromString(urlString string) (string, error)
	CreateShortURLsBatch(urlStrings []string) ([]string, error)
	GetOriginalURL(code string) (string, error)
}

// Handler обрабатывает HTTP запросы
type Handler struct {
	usecase URLUsecase
	logger  *zap.Logger
	dbPool  db.Database
}

// New создает новый экземпляр Handler
func New(usecase URLUsecase, logger *zap.Logger, dbPool db.Database) *Handler {
	return &Handler{
		usecase: usecase,
		logger:  logger,
		dbPool:  dbPool,
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
	default:
		h.logger.Error("internal server error", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	}
}
