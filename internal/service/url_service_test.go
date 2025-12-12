package service

import (
	"errors"
	"testing"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/avc-dev/url-shortener/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestCreateShortURL_Success проверяет успешное создание короткого URL
func TestCreateShortURL_Success(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	cfg := config.NewDefaultConfig()

	mockRepo.EXPECT().
		CreateOrGetCode(mock.AnythingOfType("model.URL")).
		Return("", true, nil). // true = создана новая запись, пустой код значит URL не существует
		Once()

	// Мокаем GetURLByCode - сначала возвращаем ошибку (код не найден), потом успех
	mockRepo.EXPECT().
		GetURLByCode(mock.AnythingOfType("model.Code")).
		Return(model.URL(""), store.ErrNotFound).
		Maybe() // Может быть вызвано несколько раз

	mockRepo.EXPECT().
		CreateURL(mock.AnythingOfType("model.Code"), mock.AnythingOfType("model.URL")).
		Return(nil).
		Once()

	service := NewURLService(mockRepo, cfg)
	originalURL := model.URL("https://example.com")

	// Act
	code, err := service.CreateShortURL(originalURL)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, code)
}

// TestCreateShortURL_URLAlreadyExists проверяет возврат существующего кода при дублировании URL
func TestCreateShortURL_URLAlreadyExists(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	cfg := config.NewDefaultConfig()
	existingCode := model.Code("existing")
	mockRepo.EXPECT().
		CreateOrGetCode(mock.AnythingOfType("model.URL")).
		Return(existingCode, false, nil). // false = не создана новая запись
		Once()

	service := NewURLService(mockRepo, cfg)
	originalURL := model.URL("https://example.com")

	// Act
	code, err := service.CreateShortURL(originalURL)

	// Assert
	require.Error(t, err)
	assert.True(t, errors.Is(err, store.ErrURLAlreadyExists))
	assert.Equal(t, existingCode, code)
}
