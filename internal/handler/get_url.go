package handler

import (
	"net/http"

	"github.com/avc-dev/url-shortener/internal/model"
)

func (u *Usecase) GetURL(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	code := req.URL.Path[1:]
	url, err := u.repo.GetURLByCode(model.Code(code))

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	http.Redirect(w, req, url.String(), http.StatusTemporaryRedirect)
}
