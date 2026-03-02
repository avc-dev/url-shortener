package handler

import (
	"net/http"

	"github.com/avc-dev/url-shortener/internal/audit"
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

	userID, _ := h.getUserIDFromRequest(req)
	h.emitAudit(req, audit.ActionFollow, userID, originalURL)

	http.Redirect(w, req, originalURL, http.StatusTemporaryRedirect)
}
