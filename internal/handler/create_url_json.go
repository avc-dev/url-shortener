package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/avc-dev/url-shortener/internal/usecase"
	"go.uber.org/zap"
)

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
}

// CreateURLJSON обрабатывает POST запрос для создания короткого URL (JSON формат)
func (h *Handler) CreateURLJSON(w http.ResponseWriter, req *http.Request) {
	var request ShortenRequest
	if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
		h.logger.Warn("failed to decode JSON request",
			zap.Error(err),
			zap.String("remote_addr", req.RemoteAddr),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	shortURL, err := h.usecase.CreateShortURLFromString(request.URL)
	if err != nil {
		// Проверяем, является ли ошибка дублированием URL
		var urlExistsErr usecase.URLAlreadyExistsError
		if errors.As(err, &urlExistsErr) {
			h.logger.Debug("URL already exists", zap.Error(err))
			response := ShortenResponse{
				Result: urlExistsErr.ExistingCode(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(response)
			return
		}

		h.handleError(w, err)
		return
	}

	response := ShortenResponse{
		Result: shortURL,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}
