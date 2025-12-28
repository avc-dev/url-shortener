package handler

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// GetUserURLs возвращает все URL для аутентифицированного пользователя
func (h *Handler) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.getUserIDFromRequest(r)
	if !ok {
		h.logger.Debug("user ID not found in context")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	urls, err := h.usecase.GetURLsByUserID(userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Если нет URL, возвращаем 204 No Content
	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(urls); err != nil {
		h.logger.Error("failed to encode user URLs", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
