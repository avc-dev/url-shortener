package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetStats_Success(t *testing.T) {
	mockUsecase := mocks.NewMockURLUsecase(t)
	mockUsecase.EXPECT().GetStats().Return(42, 7, nil).Once()

	h := New(mockUsecase, zap.NewNop(), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
	w := httptest.NewRecorder()

	h.GetStats(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var body statsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, 42, body.URLs)
	assert.Equal(t, 7, body.Users)
}

func TestGetStats_ZeroCounts(t *testing.T) {
	mockUsecase := mocks.NewMockURLUsecase(t)
	mockUsecase.EXPECT().GetStats().Return(0, 0, nil).Once()

	h := New(mockUsecase, zap.NewNop(), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
	w := httptest.NewRecorder()

	h.GetStats(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body statsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, 0, body.URLs)
	assert.Equal(t, 0, body.Users)
}

func TestGetStats_UsecaseError(t *testing.T) {
	mockUsecase := mocks.NewMockURLUsecase(t)
	mockUsecase.EXPECT().GetStats().Return(0, 0, errors.New("storage unavailable")).Once()

	h := New(mockUsecase, zap.NewNop(), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
	w := httptest.NewRecorder()

	h.GetStats(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
