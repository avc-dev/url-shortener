package usecase

import (
	"errors"
	"strings"
	"testing"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCreateShortURLFromString_Success(t *testing.T) {
	tests := []struct {
		name           string
		inputURL       string
		expectedURL    string
		generatedCode  string
		expectedResult string
	}{
		{
			name:           "Valid HTTP URL",
			inputURL:       "https://example.com",
			expectedURL:    "https://example.com",
			generatedCode:  "abc12345",
			expectedResult: "http://localhost:8080/abc12345",
		},
		{
			name:           "Valid URL with path",
			inputURL:       "https://example.com/path/to/resource",
			expectedURL:    "https://example.com/path/to/resource",
			generatedCode:  "xyz98765",
			expectedResult: "http://localhost:8080/xyz98765",
		},
		{
			name:           "URL with query params",
			inputURL:       "https://example.com?param=value&other=test",
			expectedURL:    "https://example.com?param=value&other=test",
			generatedCode:  "qwerty12",
			expectedResult: "http://localhost:8080/qwerty12",
		},
		{
			name:           "URL with unicode",
			inputURL:       "https://example.com/путь",
			expectedURL:    "https://example.com/путь",
			generatedCode:  "unicode1",
			expectedResult: "http://localhost:8080/unicode1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := mocks.NewMockURLRepository(t)
			mockService := mocks.NewMockURLService(t)
			cfg := config.NewDefaultConfig()

			mockService.EXPECT().
				CreateShortURL(model.URL(tt.expectedURL)).
				Return(model.Code(tt.generatedCode), nil).
				Once()

			usecase := NewURLUsecase(mockRepo, mockService, cfg, zap.NewNop())

			// Act
			result, err := usecase.CreateShortURLFromString(tt.inputURL)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestCreateShortURLFromString_URLCleaning(t *testing.T) {
	tests := []struct {
		name          string
		inputURL      string
		expectedURL   string
		generatedCode string
	}{
		{
			name:          "URL with double quotes",
			inputURL:      `"https://example.com"`,
			expectedURL:   "https://example.com",
			generatedCode: "test1234",
		},
		{
			name:          "URL with single quotes",
			inputURL:      `'https://example.com'`,
			expectedURL:   "https://example.com",
			generatedCode: "test1234",
		},
		{
			name:          "URL with spaces",
			inputURL:      "  https://example.com  ",
			expectedURL:   "https://example.com",
			generatedCode: "test1234",
		},
		{
			name:          "URL with quotes and spaces",
			inputURL:      `  "https://example.com"  `,
			expectedURL:   "https://example.com",
			generatedCode: "test1234",
		},
		{
			name:          "URL with newlines",
			inputURL:      "https://example.com\n\r",
			expectedURL:   "https://example.com",
			generatedCode: "test1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := mocks.NewMockURLRepository(t)
			mockService := mocks.NewMockURLService(t)
			cfg := config.NewDefaultConfig()

			mockService.EXPECT().
				CreateShortURL(model.URL(tt.expectedURL)).
				Return(model.Code(tt.generatedCode), nil).
				Once()

			usecase := NewURLUsecase(mockRepo, mockService, cfg, zap.NewNop())

			// Act
			result, err := usecase.CreateShortURLFromString(tt.inputURL)

			// Assert
			require.NoError(t, err)
			assert.Contains(t, result, tt.generatedCode)
		})
	}
}

func TestCreateShortURLFromString_EmptyURL(t *testing.T) {
	tests := []struct {
		name     string
		inputURL string
	}{
		{
			name:     "Completely empty",
			inputURL: "",
		},
		{
			name:     "Only spaces",
			inputURL: "   ",
		},
		{
			name:     "Only quotes",
			inputURL: `""`,
		},
		{
			name:     "Only single quotes",
			inputURL: `''`,
		},
		{
			name:     "Spaces and quotes",
			inputURL: `  ""  `,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := mocks.NewMockURLRepository(t)
			mockService := mocks.NewMockURLService(t)
			cfg := config.NewDefaultConfig()

			usecase := NewURLUsecase(mockRepo, mockService, cfg, zap.NewNop())

			// Act
			result, err := usecase.CreateShortURLFromString(tt.inputURL)

			// Assert
			assert.ErrorIs(t, err, ErrEmptyURL)
			assert.Empty(t, result)
			mockService.AssertNotCalled(t, "CreateShortURL")
		})
	}
}

func TestCreateShortURLFromString_InvalidURL(t *testing.T) {
	tests := []struct {
		name     string
		inputURL string
	}{
		{
			name:     "No scheme",
			inputURL: "example.com",
		},
		{
			name:     "No host",
			inputURL: "https://",
		},
		{
			name:     "Invalid format",
			inputURL: "not a url at all",
		},
		{
			name:     "Only path",
			inputURL: "/path/to/resource",
		},
		{
			name:     "Incomplete URL",
			inputURL: "http:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := mocks.NewMockURLRepository(t)
			mockService := mocks.NewMockURLService(t)
			cfg := config.NewDefaultConfig()

			usecase := NewURLUsecase(mockRepo, mockService, cfg, zap.NewNop())

			// Act
			result, err := usecase.CreateShortURLFromString(tt.inputURL)

			// Assert
			assert.ErrorIs(t, err, ErrInvalidURL)
			assert.Empty(t, result)
			mockService.AssertNotCalled(t, "CreateShortURL")
		})
	}
}

func TestCreateShortURLFromString_ServiceError(t *testing.T) {
	tests := []struct {
		name         string
		serviceError error
	}{
		{
			name:         "Database error",
			serviceError: errors.New("database connection failed"),
		},
		{
			name:         "Max retries exceeded",
			serviceError: errors.New("max retries exceeded"),
		},
		{
			name:         "Generic error",
			serviceError: errors.New("internal error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := mocks.NewMockURLRepository(t)
			mockService := mocks.NewMockURLService(t)
			cfg := config.NewDefaultConfig()

			mockService.EXPECT().
				CreateShortURL(model.URL("https://example.com")).
				Return(model.Code(""), tt.serviceError).
				Once()

			usecase := NewURLUsecase(mockRepo, mockService, cfg, zap.NewNop())

			// Act
			result, err := usecase.CreateShortURLFromString("https://example.com")

			// Assert
			assert.ErrorIs(t, err, ErrServiceUnavailable)
			assert.Empty(t, result)
		})
	}
}

func TestCreateShortURLFromString_LongURL(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	mockService := mocks.NewMockURLService(t)
	cfg := config.NewDefaultConfig()

	longPath := strings.Repeat("a", 2000)
	longURL := "https://example.com/" + longPath
	generatedCode := "longurl1"

	mockService.EXPECT().
		CreateShortURL(model.URL(longURL)).
		Return(model.Code(generatedCode), nil).
		Once()

	usecase := NewURLUsecase(mockRepo, mockService, cfg, zap.NewNop())

	// Act
	result, err := usecase.CreateShortURLFromString(longURL)

	// Assert
	require.NoError(t, err)
	assert.Contains(t, result, generatedCode)
}

func TestCreateShortURLFromString_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name        string
		inputURL    string
		expectedURL string
	}{
		{
			name:        "URL with anchor",
			inputURL:    "https://example.com/page#section",
			expectedURL: "https://example.com/page#section",
		},
		{
			name:        "URL with multiple query params",
			inputURL:    "https://example.com?a=1&b=2&c=3",
			expectedURL: "https://example.com?a=1&b=2&c=3",
		},
		{
			name:        "URL with encoded characters",
			inputURL:    "https://example.com/path%20with%20spaces",
			expectedURL: "https://example.com/path%20with%20spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := mocks.NewMockURLRepository(t)
			mockService := mocks.NewMockURLService(t)
			cfg := config.NewDefaultConfig()

			generatedCode := "test1234"
			mockService.EXPECT().
				CreateShortURL(model.URL(tt.expectedURL)).
				Return(model.Code(generatedCode), nil).
				Once()

			usecase := NewURLUsecase(mockRepo, mockService, cfg, zap.NewNop())

			// Act
			result, err := usecase.CreateShortURLFromString(tt.inputURL)

			// Assert
			require.NoError(t, err)
			assert.Contains(t, result, generatedCode)
		})
	}
}
