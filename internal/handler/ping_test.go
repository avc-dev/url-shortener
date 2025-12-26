package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestPing_Success(t *testing.T) {
	mockDB := mocks.NewMockDatabase(t)
	mockDB.EXPECT().Ping(mock.Anything).Return(nil).Once()

	handler := New(nil, zap.NewNop(), mockDB, nil)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	handler.Ping(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockDB.AssertExpectations(t)
}

func TestPing_DatabaseError(t *testing.T) {
	mockDB := mocks.NewMockDatabase(t)
	mockDB.EXPECT().Ping(mock.Anything).Return(assert.AnError).Once()

	handler := New(nil, zap.NewNop(), mockDB, nil)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	handler.Ping(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockDB.AssertExpectations(t)
}

func TestPing_DatabaseNotConfigured(t *testing.T) {
	handler := New(nil, zap.NewNop(), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	handler.Ping(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
