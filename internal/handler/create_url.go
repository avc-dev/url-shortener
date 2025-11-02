package handler

import (
	"io"
	"net/http"

	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/avc-dev/url-shortener/internal/service"
)

func (u *Usecase) CreateURL(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	url := string(body)
	code, err := service.GenerateCode(
		func(code string) error {
			return u.repo.CreateURL(model.Code(code), model.URL(url))
		})
	if err != nil {
		w.WriteHeader(http.StatusLoopDetected)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)

	// TODO cfg: move to config
	w.Write([]byte("http://localhost:8080/" + code))
}
