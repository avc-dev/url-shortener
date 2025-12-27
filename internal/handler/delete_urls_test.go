package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avc-dev/url-shortener/internal/middleware"
	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestDeleteURLs_Success проверяет успешное удаление URL
func TestDeleteURLs_Success(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)
	userID := "test-user"

	mockUsecase.EXPECT().
		DeleteURLs([]string{"abc123", "def456"}, userID).
		Return(nil).
		Once()

	handler := New(mockUsecase, zap.NewNop(), nil)

	codes := []string{"abc123", "def456"}
	body, _ := json.Marshal(codes)
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Добавляем userID в контекст
	ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	handler.DeleteURLs(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
}

// TestDeleteURLs_EmptyCodes проверяет обработку пустого массива кодов
func TestDeleteURLs_EmptyCodes(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)
	handler := New(mockUsecase, zap.NewNop(), nil)

	codes := []string{}
	body, _ := json.Marshal(codes)
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Добавляем userID в контекст
	ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, "test-user")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	handler.DeleteURLs(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mockUsecase.AssertNotCalled(t, "DeleteURLs")
}

// TestDeleteURLs_InvalidJSON проверяет обработку невалидного JSON
func TestDeleteURLs_InvalidJSON(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)
	handler := New(mockUsecase, zap.NewNop(), nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	// Добавляем userID в контекст
	ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, "test-user")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	handler.DeleteURLs(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mockUsecase.AssertNotCalled(t, "DeleteURLs")
}

// TestDeleteURLs_NoUserID проверяет обработку запроса без userID
func TestDeleteURLs_NoUserID(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)
	handler := New(mockUsecase, zap.NewNop(), nil)

	codes := []string{"abc123"}
	body, _ := json.Marshal(codes)
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Act
	handler.DeleteURLs(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	mockUsecase.AssertNotCalled(t, "DeleteURLs")
}

// TestDeleteURLs_UsecaseError проверяет обработку ошибки usecase
func TestDeleteURLs_UsecaseError(t *testing.T) {
	// Arrange
	mockUsecase := mocks.NewMockURLUsecase(t)
	userID := "test-user"

	mockUsecase.EXPECT().
		DeleteURLs([]string{"abc123"}, userID).
		Return(assert.AnError).
		Once()

	handler := New(mockUsecase, zap.NewNop(), nil)

	codes := []string{"abc123"}
	body, _ := json.Marshal(codes)
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Добавляем userID в контекст
	ctx := context.WithValue(req.Context(), middleware.UserIDContextKey, userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Act
	handler.DeleteURLs(w, req)

	// Assert
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
