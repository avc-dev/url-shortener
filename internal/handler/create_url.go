package handler

import (
	"io"
	"net/http"
	"net/url"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/model"
)

func (u *Usecase) CreateURL(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	originalURL := model.URL(body)

	// Генерируем уникальный код и сохраняем через service layer
	code, err := u.service.CreateShortURL(originalURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Формируем URL ответа
	shortURL, err := url.JoinPath(config.BaseURL.String(), string(code))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}
