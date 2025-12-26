package handler

import (
	"encoding/json"
	"net/http"

	"github.com/avc-dev/url-shortener/internal/model"
	"go.uber.org/zap"
)

// CreateURLBatch обрабатывает POST запрос для создания нескольких коротких URL (batch формат)
func (h *Handler) CreateURLBatch(w http.ResponseWriter, req *http.Request) {
	userID, _ := h.getUserIDFromRequest(req)
	// userID может быть пустым для анонимных пользователей

	var requests []model.BatchShortenRequest
	if err := json.NewDecoder(req.Body).Decode(&requests); err != nil {
		h.logger.Warn("failed to decode JSON request",
			zap.Error(err),
			zap.String("remote_addr", req.RemoteAddr),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Проверяем, что батч не пустой
	if len(requests) == 0 {
		h.logger.Warn("empty batch request",
			zap.String("remote_addr", req.RemoteAddr),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Извлекаем URL из запросов
	urlStrings := make([]string, len(requests))
	for i, request := range requests {
		urlStrings[i] = request.OriginalURL
	}

	// Создаем короткие URL
	shortURLs, err := h.usecase.CreateShortURLsBatch(urlStrings, userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	// Формируем ответ
	responses := make([]model.BatchShortenResponse, len(requests))
	for i, request := range requests {
		responses[i] = model.BatchShortenResponse{
			CorrelationID: request.CorrelationID,
			ShortURL:      shortURLs[i],
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(responses)
}
