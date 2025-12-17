package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/avc-dev/url-shortener/internal/usecase"
	"go.uber.org/zap"
)

// CreateURL обрабатывает POST запрос для создания короткого URL (plain text формат)
func (h *Handler) CreateURL(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		h.logger.Warn("failed to read request body",
			zap.Error(err),
			zap.String("remote_addr", req.RemoteAddr),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	shortURL, err := h.usecase.CreateShortURLFromString(string(body))
	if err != nil {
		// Проверяем, является ли ошибка дублированием URL
		var urlExistsErr usecase.URLAlreadyExistsError
		if errors.As(err, &urlExistsErr) {
			h.logger.Debug("URL already exists", zap.Error(err))
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(urlExistsErr.ExistingCode()))
			return
		}

		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}
