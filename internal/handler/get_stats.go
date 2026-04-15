package handler

import (
	"encoding/json"
	"net/http"
)

type statsResponse struct {
	URLs  int `json:"urls"`
	Users int `json:"users"`
}

// GetStats возвращает количество URL и пользователей в сервисе.
// Проверка доступа по IP выполняется middleware.TrustedSubnet на уровне роутера.
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.usecase.GetStats()
	if err != nil {
		h.logger.Error("failed to get stats")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if encErr := json.NewEncoder(w).Encode(statsResponse{URLs: stats.URLCount, Users: stats.UserCount}); encErr != nil {
		h.logger.Error("failed to encode stats response")
	}
}
