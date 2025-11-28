package handler

import (
	"encoding/json"
	"net/http"

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
