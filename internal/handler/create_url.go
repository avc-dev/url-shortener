package handler

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/avc-dev/url-shortener/internal/model"
)

func (u *Usecase) CreateURL(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Очищаем URL от пробелов и кавычек
	urlString := strings.TrimSpace(string(body))
	urlString = strings.Trim(urlString, `"'`)

	// Валидируем URL
	parsedURL, err := url.Parse(urlString)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	originalURL := model.URL(urlString)

	// Генерируем уникальный код и сохраняем через service layer
	code, err := u.service.CreateShortURL(originalURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Формируем URL ответа
	shortURL, err := url.JoinPath(u.cfg.BaseURL.String(), string(code))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}
