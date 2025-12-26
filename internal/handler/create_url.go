package handler

import (
	"io"
	"net/http"

	"go.uber.org/zap"
)

// CreateURL обрабатывает POST запрос для создания короткого URL (plain text формат)
func (h *Handler) CreateURL(w http.ResponseWriter, req *http.Request) {
	userID, ok := h.getUserIDFromRequest(req)
	if !ok {
		h.logger.Debug("user ID not found in context")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		h.logger.Warn("failed to read request body",
			zap.Error(err),
			zap.String("remote_addr", req.RemoteAddr),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	shortURL, err := h.usecase.CreateShortURLFromString(string(body), userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}
