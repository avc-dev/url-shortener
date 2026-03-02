package handler

import (
	"encoding/json"
	"net/http"

	"github.com/avc-dev/url-shortener/internal/audit"
	"go.uber.org/zap"
)

// ShortenRequest — тело POST-запроса к /api/shorten.
type ShortenRequest struct {
	// URL — оригинальный URL, который нужно сократить.
	URL string `json:"url"`
}

// ShortenResponse — тело ответа на успешный POST /api/shorten.
type ShortenResponse struct {
	// Result — полный короткий URL (например, "http://localhost:8080/AbCdEfGh").
	Result string `json:"result"`
}

// CreateURLJSON обрабатывает POST запрос для создания короткого URL (JSON формат)
func (h *Handler) CreateURLJSON(w http.ResponseWriter, req *http.Request) {
	userID, _ := h.getUserIDFromRequest(req)
	// userID может быть пустым для анонимных пользователей

	var request ShortenRequest
	if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
		h.logger.Warn("failed to decode JSON request",
			zap.Error(err),
			zap.String("remote_addr", req.RemoteAddr),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	shortURL, err := h.usecase.CreateShortURLFromString(request.URL, userID)
	if err != nil {
		h.handleErrorJSON(w, err)
		return
	}

	h.emitAudit(req, audit.ActionShorten, userID, request.URL)

	response := ShortenResponse{
		Result: shortURL,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}
