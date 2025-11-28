package usecase

import (
	"errors"
	"testing"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetOriginalURL_Success(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		storedURL   string
		expectedURL string
	}{
		{
			name:        "Simple code",
			code:        "abc12345",
			storedURL:   "https://example.com",
			expectedURL: "https://example.com",
		},
		{
			name:        "Code with URL containing path",
			code:        "xyz98765",
			storedURL:   "https://example.com/path/to/resource",
			expectedURL: "https://example.com/path/to/resource",
		},
		{
			name:        "Code with URL containing query params",
			code:        "qwerty12",
			storedURL:   "https://example.com?param=value&other=test",
			expectedURL: "https://example.com?param=value&other=test",
		},
		{
			name:        "Code with URL containing anchor",
			code:        "anchor99",
			storedURL:   "https://example.com/page#section",
			expectedURL: "https://example.com/page#section",
		},
		{
			name:        "Code with unicode URL",
			code:        "unicode1",
			storedURL:   "https://example.com/путь",
			expectedURL: "https://example.com/путь",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := mocks.NewMockURLRepository(t)
			mockService := mocks.NewMockURLService(t)
			cfg := config.NewDefaultConfig()

			mockRepo.EXPECT().
				GetURLByCode(model.Code(tt.code)).
				Return(model.URL(tt.storedURL), nil).
				Once()

			usecase := NewURLUsecase(mockRepo, mockService, cfg, zap.NewNop())

			// Act
			result, err := usecase.GetOriginalURL(tt.code)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.expectedURL, result)
		})
	}
}

func TestGetOriginalURL_NotFound(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		repoError error
	}{
		{
			name:      "Code not found",
			code:      "notexist",
			repoError: errors.New("not found"),
		},
		{
			name:      "Database error",
			code:      "dberror1",
			repoError: errors.New("database connection failed"),
		},
		{
			name:      "Empty result",
			code:      "empty123",
			repoError: errors.New("no rows"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := mocks.NewMockURLRepository(t)
			mockService := mocks.NewMockURLService(t)
			cfg := config.NewDefaultConfig()

			mockRepo.EXPECT().
				GetURLByCode(model.Code(tt.code)).
				Return(model.URL(""), tt.repoError).
				Once()

			usecase := NewURLUsecase(mockRepo, mockService, cfg, zap.NewNop())

			// Act
			result, err := usecase.GetOriginalURL(tt.code)

			// Assert
			assert.ErrorIs(t, err, ErrURLNotFound)
			assert.Empty(t, result)
		})
	}
}

func TestGetOriginalURL_EmptyCode(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	mockService := mocks.NewMockURLService(t)
	cfg := config.NewDefaultConfig()

	mockRepo.EXPECT().
		GetURLByCode(model.Code("")).
		Return(model.URL(""), errors.New("not found")).
		Once()

	usecase := NewURLUsecase(mockRepo, mockService, cfg, zap.NewNop())

	// Act
	result, err := usecase.GetOriginalURL("")

	// Assert
	assert.ErrorIs(t, err, ErrURLNotFound)
	assert.Empty(t, result)
}

func TestGetOriginalURL_SpecialCodes(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		storedURL string
	}{
		{
			name:      "Single character code",
			code:      "a",
			storedURL: "https://example.com",
		},
		{
			name:      "Very long code",
			code:      "verylongcodethatisveryverylongindeed1234567890",
			storedURL: "https://example.com",
		},
		{
			name:      "Code with numbers only",
			code:      "12345678",
			storedURL: "https://example.com",
		},
		{
			name:      "Code with uppercase letters",
			code:      "ABCDEFGH",
			storedURL: "https://example.com",
		},
		{
			name:      "Code with mixed case",
			code:      "AbCdEfGh",
			storedURL: "https://example.com",
		},
		{
			name:      "Code with special characters",
			code:      "abc-123_45",
			storedURL: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := mocks.NewMockURLRepository(t)
			mockService := mocks.NewMockURLService(t)
			cfg := config.NewDefaultConfig()

			mockRepo.EXPECT().
				GetURLByCode(model.Code(tt.code)).
				Return(model.URL(tt.storedURL), nil).
				Once()

			usecase := NewURLUsecase(mockRepo, mockService, cfg, zap.NewNop())

			// Act
			result, err := usecase.GetOriginalURL(tt.code)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.storedURL, result)
		})
	}
}
