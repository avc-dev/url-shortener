package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/avc-dev/url-shortener/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestCreateURLJSON_Success проверяет успешное создание короткого URL через JSON API
func TestCreateURLJSON_Success(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)
	expectedShortURL := "http://localhost:8080/abc12345"

	mockUsecase.EXPECT().
		CreateShortURLFromString("https://example.com", "").
		Return(expectedShortURL, nil).
		Once()

	handler := New(mockUsecase, zap.NewNop(), nil, nil)

	requestBody := ShortenRequest{URL: "https://example.com"}
	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.CreateURLJSON(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var response ShortenResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, expectedShortURL, response.Result)
}

// TestCreateURLJSON_InvalidJSON проверяет HTTP обработку невалидного JSON
func TestCreateURLJSON_InvalidJSON(t *testing.T) {
	tests := []struct {
		name        string
		requestBody string
	}{
		{
			name:        "Malformed JSON",
			requestBody: `{"url": "https://example.com"`,
		},
		{
			name:        "Invalid JSON syntax",
			requestBody: `{url: https://example.com}`,
		},
		{
			name:        "Empty body",
			requestBody: "",
		},
		{
			name:        "Not a JSON",
			requestBody: "just plain text",
		},
		{
			name:        "Array instead of object",
			requestBody: `["https://example.com"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUsecase := mocks.NewMockURLUsecase(t)
			handler := New(mockUsecase, zap.NewNop(), nil, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Act
			handler.CreateURLJSON(w, req)

			// Assert
			resp := w.Result()
			defer resp.Body.Close()

			// Проверяем что невалидный JSON возвращает BadRequest
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			// Проверяем что usecase не вызывался
			mockUsecase.AssertNotCalled(t, "CreateShortURLFromString")
		})
	}
}

// TestCreateURLJSON_ErrorMapping проверяет маппинг ошибок usecase на HTTP статусы
func TestCreateURLJSON_ErrorMapping(t *testing.T) {
	tests := []struct {
		name               string
		usecaseError       error
		expectedHTTPStatus int
	}{
		{
			name:               "ErrEmptyURL maps to 400",
			usecaseError:       usecase.ErrEmptyURL,
			expectedHTTPStatus: http.StatusBadRequest,
		},
		{
			name:               "ErrInvalidURL maps to 400",
			usecaseError:       usecase.ErrInvalidURL,
			expectedHTTPStatus: http.StatusBadRequest,
		},
		{
			name:               "ErrServiceUnavailable maps to 500",
			usecaseError:       usecase.ErrServiceUnavailable,
			expectedHTTPStatus: http.StatusInternalServerError,
		},
		{
			name:               "URLAlreadyExistsError maps to 409",
			usecaseError:       usecase.URLAlreadyExistsError{Code: "http://localhost:8080/abc123"},
			expectedHTTPStatus: http.StatusConflict,
		},
		{
			name:               "Unknown error maps to 500",
			usecaseError:       errors.New("unknown error"),
			expectedHTTPStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUsecase := mocks.NewMockURLUsecase(t)
			mockUsecase.EXPECT().
				CreateShortURLFromString("https://example.com", "").
				Return("", tt.usecaseError).
				Once()

			handler := New(mockUsecase, zap.NewNop(), nil, nil)

			requestBody := ShortenRequest{URL: "https://example.com"}
			bodyBytes, err := json.Marshal(requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Act
			handler.CreateURLJSON(w, req)

			// Assert
			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedHTTPStatus, resp.StatusCode)
		})
	}
}

// TestCreateURLJSON_ResponseFormat проверяет формат JSON ответа
func TestCreateURLJSON_ResponseFormat(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)
	expectedShortURL := "http://localhost:8080/abc12345"
	mockUsecase.EXPECT().
		CreateShortURLFromString("https://practicum.yandex.ru", "").
		Return(expectedShortURL, nil).
		Once()

	handler := New(mockUsecase, zap.NewNop(), nil, nil)

	requestBody := ShortenRequest{URL: "https://practicum.yandex.ru"}
	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.CreateURLJSON(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	// Проверяем структуру ответа
	bodyBytes, err = io.ReadAll(resp.Body)
	require.NoError(t, err)

	var response ShortenResponse
	err = json.Unmarshal(bodyBytes, &response)
	require.NoError(t, err)

	assert.Equal(t, expectedShortURL, response.Result)

	// Проверяем что в JSON есть поле "result"
	var rawResponse map[string]interface{}
	err = json.Unmarshal(bodyBytes, &rawResponse)
	require.NoError(t, err)
	assert.Contains(t, rawResponse, "result")
}

// TestCreateURLJSON_ContentType проверяет что правильный Content-Type устанавливается
func TestCreateURLJSON_ContentType(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)
	mockUsecase.EXPECT().
		CreateShortURLFromString("https://example.com", "").
		Return("http://localhost:8080/abc12345", nil).
		Once()

	handler := New(mockUsecase, zap.NewNop(), nil, nil)

	requestBody := ShortenRequest{URL: "https://example.com"}
	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(bodyBytes))
	w := httptest.NewRecorder()

	// Act
	handler.CreateURLJSON(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
}

// TestCreateURLJSON_PassesURLAsIs проверяет что handler передает URL в usecase как есть
func TestCreateURLJSON_PassesURLAsIs(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)

	// Проверяем что usecase получает URL как есть из JSON
	inputURL := "https://example.com"
	mockUsecase.EXPECT().
		CreateShortURLFromString(inputURL, "").
		Return("http://localhost:8080/test1234", nil).
		Once()

	handler := New(mockUsecase, zap.NewNop(), nil, nil)

	requestBody := ShortenRequest{URL: inputURL}
	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.CreateURLJSON(w, req)

	// Assert - usecase получил правильные данные
	mockUsecase.AssertExpectations(t)
}

// TestCreateURLJSON_URLAlreadyExists проверяет возврат существующего кода при конфликте URL
func TestCreateURLJSON_URLAlreadyExists(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)
	existingShortURL := "http://localhost:8080/existing"
	mockUsecase.EXPECT().
		CreateShortURLFromString("https://example.com", "").
		Return("", usecase.URLAlreadyExistsError{Code: existingShortURL}).
		Once()

	handler := New(mockUsecase, zap.NewNop(), nil, nil)

	requestBody := ShortenRequest{URL: "https://example.com"}
	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.CreateURLJSON(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	// Проверяем структуру ответа
	bodyBytes, err = io.ReadAll(resp.Body)
	require.NoError(t, err)

	var response ShortenResponse
	err = json.Unmarshal(bodyBytes, &response)
	require.NoError(t, err)

	assert.Equal(t, existingShortURL, response.Result)
}
