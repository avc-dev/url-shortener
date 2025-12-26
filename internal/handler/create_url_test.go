package handler

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/avc-dev/url-shortener/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestCreateURL_Success проверяет успешное создание короткого URL через plain text API
func TestCreateURL_Success(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)
	expectedShortURL := "http://localhost:8080/testcode"

	mockUsecase.EXPECT().
		CreateShortURLFromString("https://example.com", "").
		Return(expectedShortURL, nil).
		Once()

	handler := New(mockUsecase, zap.NewNop(), nil, nil)

	body := bytes.NewBufferString("https://example.com")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	// Act
	handler.CreateURL(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, expectedShortURL, string(respBody))
}

// TestCreateURL_EmptyBody проверяет HTTP обработку пустого body
func TestCreateURL_EmptyBody(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)

	// Usecase получит пустую строку и вернет ошибку валидации
	mockUsecase.EXPECT().
		CreateShortURLFromString("", "").
		Return("", usecase.ErrEmptyURL).
		Once()

	handler := New(mockUsecase, zap.NewNop(), nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(""))
	w := httptest.NewRecorder()

	// Act
	handler.CreateURL(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	// Проверяем что ошибка валидации маппится в BadRequest
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestCreateURL_ReadBodyError проверяет обработку ошибки чтения body
func TestCreateURL_ReadBodyError(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)
	handler := New(mockUsecase, zap.NewNop(), nil, nil)

	// Создаем reader который всегда возвращает ошибку
	errorReader := &errorReader{err: errors.New("read error")}
	req := httptest.NewRequest(http.MethodPost, "/", errorReader)
	w := httptest.NewRecorder()

	// Act
	handler.CreateURL(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Usecase не должен вызываться при ошибке чтения
	mockUsecase.AssertNotCalled(t, "CreateShortURLFromString")
}

// TestCreateURL_ErrorMapping проверяет маппинг ошибок usecase на HTTP статусы
func TestCreateURL_ErrorMapping(t *testing.T) {
	tests := []struct {
		name               string
		usecaseError       error
		expectedHTTPStatus int
	}{
		{
			name:               "ErrServiceUnavailable maps to 500",
			usecaseError:       usecase.ErrServiceUnavailable,
			expectedHTTPStatus: http.StatusInternalServerError,
		},
		{
			name:               "ErrInvalidURL maps to 400",
			usecaseError:       usecase.ErrInvalidURL,
			expectedHTTPStatus: http.StatusBadRequest,
		},
		{
			name:               "ErrEmptyURL maps to 400",
			usecaseError:       usecase.ErrEmptyURL,
			expectedHTTPStatus: http.StatusBadRequest,
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

			body := bytes.NewBufferString("https://example.com")
			req := httptest.NewRequest(http.MethodPost, "/", body)
			w := httptest.NewRecorder()

			// Act
			handler.CreateURL(w, req)

			// Assert
			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedHTTPStatus, resp.StatusCode)
		})
	}
}

// TestCreateURL_ContentType проверяет что правильный Content-Type устанавливается в ответе
func TestCreateURL_ContentType(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)

	mockUsecase.EXPECT().
		CreateShortURLFromString("https://example.com", "").
		Return("http://localhost:8080/testcode", nil).
		Once()

	handler := New(mockUsecase, zap.NewNop(), nil, nil)

	body := bytes.NewBufferString("https://example.com")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	// Act
	handler.CreateURL(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
}

// TestCreateURL_ResponseBody проверяет что тело ответа содержит короткий URL
func TestCreateURL_ResponseBody(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)
	expectedShortURL := "http://localhost:8080/abc12345"

	mockUsecase.EXPECT().
		CreateShortURLFromString("https://practicum.yandex.ru", "").
		Return(expectedShortURL, nil).
		Once()

	handler := New(mockUsecase, zap.NewNop(), nil, nil)

	body := bytes.NewBufferString("https://practicum.yandex.ru")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	// Act
	handler.CreateURL(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, expectedShortURL, string(respBody))
}

// TestCreateURL_PassesBodyAsIs проверяет что handler передает body в usecase как есть
func TestCreateURL_PassesBodyAsIs(t *testing.T) {
	tests := []struct {
		name     string
		inputURL string
	}{
		{
			name:     "URL with spaces",
			inputURL: "  https://example.com  ",
		},
		{
			name:     "URL with quotes",
			inputURL: `"https://example.com"`,
		},
		{
			name:     "URL with newlines",
			inputURL: "https://example.com\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUsecase := mocks.NewMockURLUsecase(t)

			// Проверяем что usecase получает URL как есть, без обработки
			mockUsecase.EXPECT().
				CreateShortURLFromString(tt.inputURL, "").
				Return("http://localhost:8080/testcode", nil).
				Once()

			handler := New(mockUsecase, zap.NewNop(), nil, nil)

			body := bytes.NewBufferString(tt.inputURL)
			req := httptest.NewRequest(http.MethodPost, "/", body)
			w := httptest.NewRecorder()

			// Act
			handler.CreateURL(w, req)

			// Assert - usecase получил правильные данные
			mockUsecase.AssertExpectations(t)
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
