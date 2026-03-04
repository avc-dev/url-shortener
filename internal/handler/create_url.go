package handler

import (
	"io"
	"net/http"

	"github.com/avc-dev/url-shortener/internal/audit"
	"go.uber.org/zap"
)

// CreateURL обрабатывает POST запрос для создания короткого URL (plain text формат)
func (h *Handler) CreateURL(w http.ResponseWriter, req *http.Request) {
	userID, _ := h.getUserIDFromRequest(req)
	// userID может быть пустым для анонимных пользователей

	body, err := io.ReadAll(req.Body)
	if err != nil {
		h.logger.Warn("failed to read request body",
			zap.Error(err),
			zap.String("remote_addr", req.RemoteAddr),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	originalURL := string(body)
	shortURL, err := h.usecase.CreateShortURLFromString(originalURL, userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.emitAudit(req, audit.ActionShorten, userID, originalURL)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}
