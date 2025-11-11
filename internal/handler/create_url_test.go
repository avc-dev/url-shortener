package handler

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateURL_Success проверяет успешное создание короткого URL
func TestCreateURL_Success(t *testing.T) {
	tests := []struct {
		name           string
		originalURL    string
		expectedStatus int
		expectedPrefix string
	}{
		{
			name:           "Valid HTTP URL",
			originalURL:    "https://example.com",
			expectedStatus: http.StatusCreated,
			expectedPrefix: config.BaseURL.String(),
		},
		{
			name:           "Valid HTTPS URL with path",
			originalURL:    "https://example.com/some/path?query=param",
			expectedStatus: http.StatusCreated,
			expectedPrefix: config.BaseURL.String(),
		},
		{
			name:           "Long URL",
			originalURL:    "https://example.com/very/long/path/that/goes/on/and/on/with/many/segments",
			expectedStatus: http.StatusCreated,
			expectedPrefix: config.BaseURL.String(),
		},
		{
			name:           "URL with special characters",
			originalURL:    "https://example.com/path?param=value&other=test#anchor",
			expectedStatus: http.StatusCreated,
			expectedPrefix: config.BaseURL.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := mocks.NewMockURLRepository(t)
			mockService := mocks.NewMockURLService(t)

			generatedCode := model.Code("testcode")

			mockService.EXPECT().
				CreateShortURL(model.URL(tt.originalURL)).
				Return(generatedCode, nil).
				Once()

			usecase := New(mockRepo, mockService)

			body := bytes.NewBufferString(tt.originalURL)
			req := httptest.NewRequest(http.MethodPost, "/", body)
			w := httptest.NewRecorder()

			// Act
			usecase.CreateURL(w, req)

			// Assert
			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			respStr := string(respBody)
			assert.True(t, strings.HasPrefix(respStr, tt.expectedPrefix),
				"Expected response to start with %s, got %s", tt.expectedPrefix, respStr)

			// Проверяем что ответ содержит код (должен быть префикс + 8 символов)
			assert.Equal(t, len(tt.expectedPrefix)+8, len(respStr))

			// Проверяем Content-Type
			assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
		})
	}
}

// TestCreateURL_EmptyBody проверяет обработку пустого body
func TestCreateURL_EmptyBody(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	mockService := mocks.NewMockURLService(t)

	generatedCode := model.Code("testcode")

	mockService.EXPECT().
		CreateShortURL(model.URL("")).
		Return(generatedCode, nil).
		Once()

	usecase := New(mockRepo, mockService)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(""))
	w := httptest.NewRecorder()

	// Act
	usecase.CreateURL(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	// Хендлер все равно должен вернуть StatusCreated даже с пустым URL
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

// TestCreateURL_ReadBodyError проверяет обработку ошибки чтения body
func TestCreateURL_ReadBodyError(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	mockService := mocks.NewMockURLService(t)
	usecase := New(mockRepo, mockService)

	// Создаем reader который всегда возвращает ошибку
	errorReader := &errorReader{err: errors.New("read error")}
	req := httptest.NewRequest(http.MethodPost, "/", errorReader)
	w := httptest.NewRecorder()

	// Act
	usecase.CreateURL(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Service и repo не должны вызываться при ошибке чтения
	mockService.AssertNotCalled(t, "CreateShortURL")
}

// TestCreateURL_ServiceError проверяет обработку ошибки от service
func TestCreateURL_ServiceError(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	mockService := mocks.NewMockURLService(t)

	// Service возвращает ошибку (может быть из-за коллизий или ошибок БД)
	mockService.EXPECT().
		CreateShortURL(model.URL("https://example.com")).
		Return(model.Code(""), errors.New("could not generate unique code")).
		Once()

	usecase := New(mockRepo, mockService)

	body := bytes.NewBufferString("https://example.com")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	// Act
	usecase.CreateURL(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// TestCreateURL_ServiceVariousErrors проверяет обработку различных ошибок от service
func TestCreateURL_ServiceVariousErrors(t *testing.T) {
	tests := []struct {
		name           string
		serviceError   error
		expectedStatus int
	}{
		{
			name:           "Database error from service",
			serviceError:   errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Connection timeout from service",
			serviceError:   errors.New("connection timeout"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Max retries exceeded",
			serviceError:   errors.New("max retries exceeded"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := mocks.NewMockURLRepository(t)
			mockService := mocks.NewMockURLService(t)

			mockService.EXPECT().
				CreateShortURL(model.URL("https://example.com")).
				Return(model.Code(""), tt.serviceError).
				Once()

			usecase := New(mockRepo, mockService)

			body := bytes.NewBufferString("https://example.com")
			req := httptest.NewRequest(http.MethodPost, "/", body)
			w := httptest.NewRecorder()

			// Act
			usecase.CreateURL(w, req)

			// Assert
			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// TestCreateURL_BoundaryConditions проверяет граничные условия
func TestCreateURL_BoundaryConditions(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "Single character URL",
			url:            "a",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Very long URL (2000+ chars)",
			url:            "https://example.com/" + strings.Repeat("a", 2000),
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "URL with newlines",
			url:            "https://example.com\n\r",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "URL with spaces",
			url:            "https://example.com/path with spaces",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Unicode URL",
			url:            "https://example.com/путь/до/ресурса",
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := mocks.NewMockURLRepository(t)
			mockService := mocks.NewMockURLService(t)

			generatedCode := model.Code("testcode")

			mockService.EXPECT().
				CreateShortURL(model.URL(tt.url)).
				Return(generatedCode, nil).
				Once()

			usecase := New(mockRepo, mockService)

			body := bytes.NewBufferString(tt.url)
			req := httptest.NewRequest(http.MethodPost, "/", body)
			w := httptest.NewRecorder()

			// Act
			usecase.CreateURL(w, req)

			// Assert
			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// errorReader - вспомогательный тип для тестирования ошибок чтения
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}
