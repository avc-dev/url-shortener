package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/handler"
	"github.com/avc-dev/url-shortener/internal/middleware"
	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/avc-dev/url-shortener/internal/repository"
	"github.com/avc-dev/url-shortener/internal/service"
	"github.com/avc-dev/url-shortener/internal/store"
	"github.com/avc-dev/url-shortener/internal/usecase"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// newTestHandler собирает хендлер поверх реального in-memory хранилища.
// Используется во всех примерах для демонстрации работы API.
func newTestHandler() *handler.Handler {
	cfg := config.NewDefaultConfig()
	st := store.NewStore()
	repo := repository.New(st)
	svc := service.NewURLService(repo, cfg)
	uc := usecase.NewURLUsecase(repo, svc, cfg, zap.NewNop())
	return handler.New(uc, zap.NewNop(), nil)
}

// withUserContext добавляет userID в контекст запроса так, как это делает AuthMiddleware.
func withUserContext(req *http.Request, userID string) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, userID)
	return req.WithContext(ctx)
}

// withChiParam добавляет chi-параметр маршрута {id} в контекст запроса.
func withChiParam(req *http.Request, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// ExampleHandler_CreateURL демонстрирует создание короткого URL через plain-text API.
//
// Эндпоинт: POST /
// Тело запроса: оригинальный URL в виде plain text.
// Ответ: 201 Created, тело — полный короткий URL.
func ExampleHandler_CreateURL() {
	h := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	w := httptest.NewRecorder()
	h.CreateURL(w, req)

	res := w.Result()
	body, _ := io.ReadAll(res.Body)

	fmt.Println(res.StatusCode)
	fmt.Println(res.Header.Get("Content-Type"))
	fmt.Println(strings.HasPrefix(string(body), "http://localhost:8080/"))
	// Output:
	// 201
	// text/plain
	// true
}

// ExampleHandler_CreateURLJSON демонстрирует создание короткого URL через JSON API.
//
// Эндпоинт: POST /api/shorten
// Тело запроса: {"url": "..."}.
// Ответ: 201 Created, тело — {"result": "полный_короткий_url"}.
func ExampleHandler_CreateURLJSON() {
	h := newTestHandler()

	body := `{"url":"https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateURLJSON(w, req)

	res := w.Result()
	var resp handler.ShortenResponse
	json.NewDecoder(res.Body).Decode(&resp)

	fmt.Println(res.StatusCode)
	fmt.Println(strings.HasPrefix(resp.Result, "http://localhost:8080/"))
	// Output:
	// 201
	// true
}

// ExampleHandler_CreateURLJSON_conflict демонстрирует ответ 409 при повторной отправке того же URL.
//
// При повторном запросе с тем же оригинальным URL сервис возвращает 409 Conflict
// и в теле ответа — уже существующий короткий URL.
func ExampleHandler_CreateURLJSON_conflict() {
	h := newTestHandler()

	body := `{"url":"https://example.com"}`

	// Первый запрос — успешное создание (201)
	req1 := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	h.CreateURLJSON(w1, req1)

	// Второй запрос — тот же URL, ожидаем 409 Conflict
	req2 := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	h.CreateURLJSON(w2, req2)

	fmt.Println(w1.Code)
	fmt.Println(w2.Code)
	// Output:
	// 201
	// 409
}

// ExampleHandler_CreateURLBatch демонстрирует пакетное создание коротких URL.
//
// Эндпоинт: POST /api/shorten/batch
// Тело запроса: массив объектов {"correlation_id": "...", "original_url": "..."}.
// Ответ: 201 Created, массив {"correlation_id": "...", "short_url": "..."}.
func ExampleHandler_CreateURLBatch() {
	h := newTestHandler()

	batch := []model.BatchShortenRequest{
		{CorrelationID: "1", OriginalURL: "https://example.com"},
		{CorrelationID: "2", OriginalURL: "https://golang.org"},
	}
	data, _ := json.Marshal(batch)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.CreateURLBatch(w, req)

	res := w.Result()
	var responses []model.BatchShortenResponse
	json.NewDecoder(res.Body).Decode(&responses)

	fmt.Println(res.StatusCode)
	fmt.Println(len(responses))
	fmt.Println(responses[0].CorrelationID)
	fmt.Println(strings.HasPrefix(responses[0].ShortURL, "http://localhost:8080/"))
	// Output:
	// 201
	// 2
	// 1
	// true
}

// ExampleHandler_GetURL демонстрирует редирект по короткому коду.
//
// Эндпоинт: GET /{id}
// Ответ: 307 Temporary Redirect, заголовок Location — оригинальный URL.
func ExampleHandler_GetURL() {
	h := newTestHandler()

	// Сначала создаём короткий URL
	createReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	createW := httptest.NewRecorder()
	h.CreateURL(createW, createReq)
	shortURL := strings.TrimSpace(createW.Body.String())
	// Извлекаем код из конца URL: "http://localhost:8080/AbCdEfGh" → "AbCdEfGh"
	code := shortURL[strings.LastIndex(shortURL, "/")+1:]

	// Переходим по коду — ожидаем редирект 307
	getReq := httptest.NewRequest(http.MethodGet, "/"+code, nil)
	getReq = withChiParam(getReq, code)
	getW := httptest.NewRecorder()
	h.GetURL(getW, getReq)

	res := getW.Result()
	fmt.Println(res.StatusCode)
	fmt.Println(res.Header.Get("Location"))
	// Output:
	// 307
	// https://example.com
}

// ExampleHandler_GetURL_notFound демонстрирует ответ 404 при несуществующем коде.
func ExampleHandler_GetURL_notFound() {
	h := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	req = withChiParam(req, "nonexistent")
	w := httptest.NewRecorder()
	h.GetURL(w, req)

	fmt.Println(w.Code)
	// Output:
	// 404
}

// ExampleHandler_GetUserURLs демонстрирует получение всех URL аутентифицированного пользователя.
//
// Эндпоинт: GET /api/user/urls
// Требует аутентификации (user_id в контексте).
// Ответ: 200 OK и JSON-массив, либо 204 No Content если URL нет.
func ExampleHandler_GetUserURLs() {
	h := newTestHandler()
	const userID = "user-42"

	// Создаём URL от имени пользователя
	createReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	createReq = withUserContext(createReq, userID)
	createW := httptest.NewRecorder()
	h.CreateURL(createW, createReq)

	// Запрашиваем список URL пользователя
	listReq := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	listReq = withUserContext(listReq, userID)
	listW := httptest.NewRecorder()
	h.GetUserURLs(listW, listReq)

	res := listW.Result()
	var urls []model.UserURLResponse
	json.NewDecoder(res.Body).Decode(&urls)

	fmt.Println(res.StatusCode)
	fmt.Println(len(urls))
	fmt.Println(urls[0].OriginalURL)
	// Output:
	// 200
	// 1
	// https://example.com
}

// ExampleHandler_GetUserURLs_empty демонстрирует 204 No Content для нового пользователя без URL.
func ExampleHandler_GetUserURLs_empty() {
	h := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req = withUserContext(req, "new-user-with-no-urls")
	w := httptest.NewRecorder()
	h.GetUserURLs(w, req)

	fmt.Println(w.Code)
	// Output:
	// 204
}

// ExampleHandler_GetUserURLs_unauthorized демонстрирует 401 при отсутствии аутентификации.
func ExampleHandler_GetUserURLs_unauthorized() {
	h := newTestHandler()

	// Запрос без user_id в контексте
	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	w := httptest.NewRecorder()
	h.GetUserURLs(w, req)

	fmt.Println(w.Code)
	// Output:
	// 401
}

// ExampleHandler_DeleteURLs демонстрирует асинхронное удаление коротких URL пользователя.
//
// Эндпоинт: DELETE /api/user/urls
// Тело запроса: JSON-массив коротких кодов.
// Ответ: 202 Accepted — запрос принят, удаление выполняется асинхронно.
func ExampleHandler_DeleteURLs() {
	h := newTestHandler()
	const userID = "user-42"

	// Создаём URL, который затем удалим
	createReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	createReq = withUserContext(createReq, userID)
	createW := httptest.NewRecorder()
	h.CreateURL(createW, createReq)
	shortURL := strings.TrimSpace(createW.Body.String())
	code := shortURL[strings.LastIndex(shortURL, "/")+1:]

	// Удаляем URL — ожидаем 202 Accepted (асинхронное удаление)
	codes, _ := json.Marshal([]string{code})
	delReq := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader(codes))
	delReq = withUserContext(delReq, userID)
	w := httptest.NewRecorder()
	h.DeleteURLs(w, delReq)

	fmt.Println(w.Code)
	// Output:
	// 202
}

// ExampleHandler_Ping_noDB демонстрирует ответ 500 когда база данных не настроена.
//
// Эндпоинт: GET /ping
// Возвращает 200 если БД доступна, 500 если не настроена или недоступна.
func ExampleHandler_Ping_noDB() {
	h := newTestHandler() // dbPool == nil

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	h.Ping(w, req)

	fmt.Println(w.Code)
	// Output:
	// 500
}
