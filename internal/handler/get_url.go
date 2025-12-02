package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// GetURL обрабатывает GET запрос для редиректа на оригинальный URL по короткому коду
func (h *Handler) GetURL(w http.ResponseWriter, req *http.Request) {
	code := chi.URLParam(req, "id")

	originalURL, err := h.usecase.GetOriginalURL(code)
	if err != nil {
		h.handleError(w, err)
		return
	}

	http.Redirect(w, req, originalURL, http.StatusTemporaryRedirect)
}
