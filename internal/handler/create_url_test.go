package handler

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRepository - мок для URLRepository
type MockRepository struct {
	CreateURLFunc    func(code model.Code, url model.URL) error
	GetURLByCodeFunc func(code model.Code) (model.URL, error)
}

func (m *MockRepository) CreateURL(code model.Code, url model.URL) error {
	if m.CreateURLFunc != nil {
		return m.CreateURLFunc(code, url)
	}
	return nil
}

func (m *MockRepository) GetURLByCode(code model.Code) (model.URL, error) {
	if m.GetURLByCodeFunc != nil {
		return m.GetURLByCodeFunc(code)
	}
	return "", nil
}

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
			expectedPrefix: "http://localhost:8080/",
		},
		{
			name:           "Valid HTTPS URL with path",
			originalURL:    "https://example.com/some/path?query=param",
			expectedStatus: http.StatusCreated,
			expectedPrefix: "http://localhost:8080/",
		},
		{
			name:           "Long URL",
			originalURL:    "https://example.com/very/long/path/that/goes/on/and/on/with/many/segments",
			expectedStatus: http.StatusCreated,
			expectedPrefix: "http://localhost:8080/",
		},
		{
			name:           "URL with special characters",
			originalURL:    "https://example.com/path?param=value&other=test#anchor",
			expectedStatus: http.StatusCreated,
			expectedPrefix: "http://localhost:8080/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := &MockRepository{
				CreateURLFunc: func(code model.Code, url model.URL) error {
					assert.Equal(t, model.URL(tt.originalURL), url)
					return nil
				},
			}

			usecase := New(mockRepo)

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

// TestCreateURL_WrongMethod проверяет обработку неправильного HTTP метода
func TestCreateURL_WrongMethod(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "GET method",
			method:         http.MethodGet,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "PUT method",
			method:         http.MethodPut,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "DELETE method",
			method:         http.MethodDelete,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "PATCH method",
			method:         http.MethodPatch,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "HEAD method",
			method:         http.MethodHead,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := &MockRepository{}
			usecase := New(mockRepo)

			body := bytes.NewBufferString("https://example.com")
			req := httptest.NewRequest(tt.method, "/", body)
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

// TestCreateURL_EmptyBody проверяет обработку пустого body
func TestCreateURL_EmptyBody(t *testing.T) {
	// Arrange
	mockRepo := &MockRepository{
		CreateURLFunc: func(code model.Code, url model.URL) error {
			assert.Equal(t, model.URL(""), url)
			return nil
		},
	}

	usecase := New(mockRepo)

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
	mockRepo := &MockRepository{}
	usecase := New(mockRepo)

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
}

// TestCreateURL_GenerateCodeError проверяет обработку ошибки генерации кода
func TestCreateURL_GenerateCodeError(t *testing.T) {
	// Arrange
	callCount := 0
	mockRepo := &MockRepository{
		CreateURLFunc: func(code model.Code, url model.URL) error {
			callCount++
			// Всегда возвращаем ошибку, чтобы исчерпать все попытки
			return errors.New("duplicate code")
		},
	}

	usecase := New(mockRepo)

	body := bytes.NewBufferString("https://example.com")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	// Act
	usecase.CreateURL(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusLoopDetected, resp.StatusCode)

	// Проверяем что было сделано MaxTries попыток (100)
	assert.Equal(t, 100, callCount)
}

// TestCreateURL_RepositoryError проверяет обработку других ошибок репозитория
func TestCreateURL_RepositoryError(t *testing.T) {
	tests := []struct {
		name           string
		repoError      error
		failOnAttempt  int
		expectedStatus int
	}{
		{
			name:           "Repository error on first attempt",
			repoError:      errors.New("database error"),
			failOnAttempt:  1,
			expectedStatus: http.StatusLoopDetected,
		},
		{
			name:           "Repository error after several attempts",
			repoError:      errors.New("connection timeout"),
			failOnAttempt:  50,
			expectedStatus: http.StatusLoopDetected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			attemptCount := 0
			mockRepo := &MockRepository{
				CreateURLFunc: func(code model.Code, url model.URL) error {
					attemptCount++
					return tt.repoError
				},
			}

			usecase := New(mockRepo)

			body := bytes.NewBufferString("https://example.com")
			req := httptest.NewRequest(http.MethodPost, "/", body)
			w := httptest.NewRecorder()

			// Act
			usecase.CreateURL(w, req)

			// Assert
			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Greater(t, attemptCount, 0, "Expected at least one attempt to create URL")
		})
	}
}

// TestCreateURL_SuccessAfterRetries проверяет успех после нескольких неудачных попыток
func TestCreateURL_SuccessAfterRetries(t *testing.T) {
	// Arrange
	attemptCount := 0
	successOnAttempt := 5

	mockRepo := &MockRepository{
		CreateURLFunc: func(code model.Code, url model.URL) error {
			attemptCount++
			if attemptCount < successOnAttempt {
				// Первые несколько попыток возвращают ошибку (код уже существует)
				return errors.New("duplicate code")
			}
			// Пятая попытка успешна
			return nil
		},
	}

	usecase := New(mockRepo)

	body := bytes.NewBufferString("https://example.com")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	// Act
	usecase.CreateURL(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, successOnAttempt, attemptCount)
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
			var capturedURL model.URL
			mockRepo := &MockRepository{
				CreateURLFunc: func(code model.Code, url model.URL) error {
					capturedURL = url
					return nil
				},
			}

			usecase := New(mockRepo)

			body := bytes.NewBufferString(tt.url)
			req := httptest.NewRequest(http.MethodPost, "/", body)
			w := httptest.NewRecorder()

			// Act
			usecase.CreateURL(w, req)

			// Assert
			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Equal(t, tt.url, string(capturedURL))
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
