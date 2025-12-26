package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/avc-dev/url-shortener/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestCreateURLBatch(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockUsecase := &mocks.MockURLUsecase{}
	mockDB := &mocks.MockDatabase{}
	mockAuthService := &service.AuthService{} // Using nil for auth service as it's not tested here

	handler := New(mockUsecase, logger, mockDB, mockAuthService)

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func()
		expectedStatus int
		expectedBody   bool
	}{
		{
			name: "successful batch creation",
			requestBody: []model.BatchShortenRequest{
				{CorrelationID: "1", OriginalURL: "https://example.com"},
				{CorrelationID: "2", OriginalURL: "https://google.com"},
			},
			mockSetup: func() {
				mockUsecase.EXPECT().CreateShortURLsBatch([]string{"https://example.com", "https://google.com"}, "").
					Return([]string{"http://localhost:8080/abc123", "http://localhost:8080/def456"}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   true,
		},
		{
			name:           "empty batch",
			requestBody:    []model.BatchShortenRequest{},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   false,
		},
		{
			name:           "invalid json",
			requestBody:    "invalid json",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			handler.CreateURLBatch(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody {
				var response []model.BatchShortenResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, len(tt.requestBody.([]model.BatchShortenRequest)))
			}

			mockUsecase.AssertExpectations(t)
		})
	}
}
