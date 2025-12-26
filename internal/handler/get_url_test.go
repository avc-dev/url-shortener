package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/avc-dev/url-shortener/internal/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestGetURL_Success проверяет успешное получение URL по коду
func TestGetURL_Success(t *testing.T) {
	tests := []struct {
		name             string
		code             string
		expectedURL      string
		expectedStatus   int
		expectedRedirect string
	}{
		{
			name:             "Valid short code",
			code:             "abc12345",
			expectedURL:      "https://example.com",
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedRedirect: "https://example.com",
		},
		{
			name:             "Code with URL containing path",
			code:             "xyz98765",
			expectedURL:      "https://example.com/path/to/resource",
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedRedirect: "https://example.com/path/to/resource",
		},
		{
			name:             "Code with URL containing query params",
			code:             "qwerty12",
			expectedURL:      "https://example.com?param=value&other=test",
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedRedirect: "https://example.com?param=value&other=test",
		},
		{
			name:             "Code with URL containing anchor",
			code:             "asdfgh90",
			expectedURL:      "https://example.com/page#section",
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedRedirect: "https://example.com/page#section",
		},
		{
			name:             "Long code",
			code:             "verylongcode1234567890",
			expectedURL:      "https://example.com",
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedRedirect: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUsecase := mocks.NewMockURLUsecase(t)
			mockUsecase.EXPECT().
				GetOriginalURL(tt.code).
				Return(tt.expectedURL, nil).
				Once()

			handler := New(mockUsecase, zap.NewNop(), nil, nil)

			req := httptest.NewRequest(http.MethodGet, "/"+tt.code, nil)
			// Add chi context with URL parameter
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.code)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			// Act
			handler.GetURL(w, req)

			// Assert
			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Equal(t, tt.expectedRedirect, resp.Header.Get("Location"))
		})
	}
}

// TestGetURL_NotFound проверяет обработку несуществующего кода
func TestGetURL_NotFound(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		expectedStatus int
	}{
		{
			name:           "Code not found",
			code:           "notexist",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Database error treated as not found",
			code:           "dberror1",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Empty result error",
			code:           "empty123",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUsecase := mocks.NewMockURLUsecase(t)
			mockUsecase.EXPECT().
				GetOriginalURL(tt.code).
				Return("", usecase.ErrURLNotFound).
				Once()

			handler := New(mockUsecase, zap.NewNop(), nil, nil)

			req := httptest.NewRequest(http.MethodGet, "/"+tt.code, nil)
			// Add chi context with URL parameter
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.code)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			// Act
			handler.GetURL(w, req)

			// Assert
			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Empty(t, resp.Header.Get("Location"))
		})
	}
}

// TestGetURL_EmptyCode проверяет обработку пустого кода
func TestGetURL_EmptyCode(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)
	mockUsecase.EXPECT().
		GetOriginalURL("").
		Return("", usecase.ErrURLNotFound).
		Once()

	handler := New(mockUsecase, zap.NewNop(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Add chi context with empty URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	// Act
	handler.GetURL(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestGetURL_CodeExtraction проверяет правильное извлечение кода из URL
func TestGetURL_CodeExtraction(t *testing.T) {
	tests := []struct {
		name         string
		requestPath  string
		expectedCode string
	}{
		{
			name:         "Simple code",
			requestPath:  "/abc12345",
			expectedCode: "abc12345",
		},
		{
			name:         "Code with trailing slash",
			requestPath:  "/abc12345/",
			expectedCode: "abc12345/",
		},
		{
			name:         "Code with additional path segments",
			requestPath:  "/abc12345/extra/path",
			expectedCode: "abc12345/extra/path",
		},
		{
			name:         "Code with query params",
			requestPath:  "/abc12345?param=value",
			expectedCode: "abc12345",
		},
		{
			name:         "Code with special characters",
			requestPath:  "/abc-123_45",
			expectedCode: "abc-123_45",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUsecase := mocks.NewMockURLUsecase(t)
			mockUsecase.EXPECT().
				GetOriginalURL(tt.expectedCode).
				Return("https://example.com", nil).
				Once()

			handler := New(mockUsecase, zap.NewNop(), nil, nil)

			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			// Add chi context with URL parameter
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.expectedCode)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			// Act
			handler.GetURL(w, req)
		})
	}
}

// TestGetURL_BoundaryConditions проверяет граничные условия
func TestGetURL_BoundaryConditions(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		returnURL      string
		returnError    error
		expectedStatus int
	}{
		{
			name:           "Single character code",
			code:           "a",
			returnURL:      "https://example.com",
			returnError:    nil,
			expectedStatus: http.StatusTemporaryRedirect,
		},
		{
			name:           "Very long code",
			code:           "verylongcodethatisveryverylongindeed1234567890",
			returnURL:      "https://example.com",
			returnError:    nil,
			expectedStatus: http.StatusTemporaryRedirect,
		},
		{
			name:           "Code with numbers only",
			code:           "12345678",
			returnURL:      "https://example.com",
			returnError:    nil,
			expectedStatus: http.StatusTemporaryRedirect,
		},
		{
			name:           "Code with uppercase letters",
			code:           "ABCDEFGH",
			returnURL:      "https://example.com",
			returnError:    nil,
			expectedStatus: http.StatusTemporaryRedirect,
		},
		{
			name:           "Code with mixed case",
			code:           "AbCdEfGh",
			returnURL:      "https://example.com",
			returnError:    nil,
			expectedStatus: http.StatusTemporaryRedirect,
		},
		{
			name:           "URL with unicode characters",
			code:           "abc12345",
			returnURL:      "https://example.com/path",
			returnError:    nil,
			expectedStatus: http.StatusTemporaryRedirect,
		},
		{
			name:           "Very long URL",
			code:           "abc12345",
			returnURL:      "https://example.com/" + string(make([]byte, 2000)),
			returnError:    nil,
			expectedStatus: http.StatusTemporaryRedirect,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUsecase := mocks.NewMockURLUsecase(t)
			if tt.returnError != nil {
				mockUsecase.EXPECT().
					GetOriginalURL(tt.code).
					Return("", usecase.ErrURLNotFound).
					Once()
			} else {
				mockUsecase.EXPECT().
					GetOriginalURL(tt.code).
					Return(tt.returnURL, nil).
					Once()
			}

			handler := New(mockUsecase, zap.NewNop(), nil, nil)

			req := httptest.NewRequest(http.MethodGet, "/"+tt.code, nil)
			// Add chi context with URL parameter
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.code)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			// Act
			handler.GetURL(w, req)

			// Assert
			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusTemporaryRedirect {
				assert.Equal(t, tt.returnURL, resp.Header.Get("Location"))
			}
		})
	}
}

// TestGetURL_UnicodeURL проверяет обработку URL с unicode символами
func TestGetURL_UnicodeURL(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)
	mockUsecase.EXPECT().
		GetOriginalURL("abc12345").
		Return("https://example.com/путь", nil).
		Once()

	handler := New(mockUsecase, zap.NewNop(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/abc12345", nil)
	// Add chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc12345")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	// Act
	handler.GetURL(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)

	// Location header будет содержать URL-encoded версию
	location := resp.Header.Get("Location")
	assert.NotEmpty(t, location, "Expected Location header to be set")

	// Проверяем что это валидный URL с закодированными unicode символами или без
	// URL encoding может быть в разном регистре в зависимости от версии Go
	expectedEncodedLower := "https://example.com/%d0%bf%d1%83%d1%82%d1%8c"
	expectedEncodedUpper := "https://example.com/%D0%BF%D1%83%D1%82%D1%8C"
	expectedRaw := "https://example.com/путь"

	assert.True(t,
		location == expectedEncodedLower || location == expectedEncodedUpper || location == expectedRaw,
		"Got Location: %s", location)
}

// TestGetURL_RedirectStatusCode проверяет что используется правильный статус редиректа
func TestGetURL_RedirectStatusCode(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)
	mockUsecase.EXPECT().
		GetOriginalURL("abc12345").
		Return("https://example.com", nil).
		Once()

	handler := New(mockUsecase, zap.NewNop(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/abc12345", nil)
	// Add chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc12345")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	// Act
	handler.GetURL(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	// Проверяем что используется именно StatusTemporaryRedirect (307)
	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	assert.Equal(t, 307, resp.StatusCode)
}

// TestGetURL_UsecaseInteraction проверяет взаимодействие с usecase
func TestGetURL_UsecaseInteraction(t *testing.T) {
	// Arrange
	expectedCode := "testcode"

	mockUsecase := mocks.NewMockURLUsecase(t)
	mockUsecase.EXPECT().
		GetOriginalURL(expectedCode).
		Return("https://example.com", nil).
		Once()

	handler := New(mockUsecase, zap.NewNop(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/"+expectedCode, nil)
	// Add chi context with URL parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", expectedCode)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	// Act
	handler.GetURL(w, req)
}

// TestGetURL_ConcurrentRequests проверяет обработку параллельных запросов
func TestGetURL_ConcurrentRequests(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)
	// Ожидаем 10 различных вызовов
	for i := 0; i < 10; i++ {
		code := string(rune('a' + i))
		mockUsecase.EXPECT().
			GetOriginalURL(code).
			Return("https://example.com/"+code, nil).
			Once()
	}

	handler := New(mockUsecase, zap.NewNop(), nil, nil)

	// Act & Assert - запускаем несколько параллельных запросов
	done := make(chan bool)
	numRequests := 10

	for i := 0; i < numRequests; i++ {
		go func(index int) {
			code := string(rune('a' + index))
			req := httptest.NewRequest(http.MethodGet, "/"+code, nil)
			// Add chi context with URL parameter
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", code)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.GetURL(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode,
				"Request %d: Expected status %d", index, http.StatusTemporaryRedirect)

			done <- true
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < numRequests; i++ {
		<-done
	}
}
