package handler

import (
	"errors"
	"net/http"

	"github.com/avc-dev/url-shortener/internal/usecase"
	"go.uber.org/zap"
)

// URLUsecase определяет интерфейс для бизнес-логики работы с URL
type URLUsecase interface {
	CreateShortURLFromString(urlString string) (string, error)
	GetOriginalURL(code string) (string, error)
}

// Handler обрабатывает HTTP запросы
type Handler struct {
	usecase URLUsecase
	logger  *zap.Logger
}

// New создает новый экземпляр Handler
func New(usecase URLUsecase, logger *zap.Logger) *Handler {
	return &Handler{
		usecase: usecase,
		logger:  logger,
	}
}

// handleError маппит ошибки usecase на HTTP статусы
func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidURL), errors.Is(err, usecase.ErrEmptyURL):
		h.logger.Warn("bad request", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
	case errors.Is(err, usecase.ErrURLNotFound):
		h.logger.Debug("URL not found", zap.Error(err))
		w.WriteHeader(http.StatusNotFound)
	default:
		h.logger.Error("internal server error", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	}
}
