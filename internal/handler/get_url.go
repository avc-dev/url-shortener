package handler

import (
	"net/http"

	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/go-chi/chi/v5"
)

func (u *Usecase) GetURL(w http.ResponseWriter, req *http.Request) {
	code := chi.URLParam(req, "id")
	url, err := u.repo.GetURLByCode(model.Code(code))

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	http.Redirect(w, req, url.String(), http.StatusTemporaryRedirect)
}
