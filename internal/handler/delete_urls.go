package handler

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// DeleteURLs обрабатывает DELETE запрос для удаления нескольких URL пользователя
func (h *Handler) DeleteURLs(w http.ResponseWriter, req *http.Request) {
	// Получаем userID из контекста
	userID, ok := h.getUserIDFromRequest(req)
	if !ok {
		h.logger.Error("user ID not found in context")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Декодируем тело запроса
	var codes []string
	if err := json.NewDecoder(req.Body).Decode(&codes); err != nil {
		h.logger.Error("failed to decode request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Валидируем входные данные
	if len(codes) == 0 {
		h.logger.Debug("empty codes list")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Выполняем асинхронное удаление
	err := h.usecase.DeleteURLs(codes, userID)
	if err != nil {
		h.logger.Error("failed to initiate URL deletion", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Возвращаем 202 Accepted - запрос принят для обработки
	w.WriteHeader(http.StatusAccepted)
}
