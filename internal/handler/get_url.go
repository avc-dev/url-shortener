package handler

import (
	"net/http"
)

func GetUrl(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	http.Redirect(w, req, "https://practicum.yandex.ru/", http.StatusTemporaryRedirect)
}
