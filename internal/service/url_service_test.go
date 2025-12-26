package service

import (
	"testing"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateShortURL_Success проверяет успешное создание короткого URL
func TestCreateShortURL_Success(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	mockGenerator := mocks.NewMockGenerator(t)
	expectedCode := model.Code("abc123")

	// Генератор возвращает ожидаемый код
	mockGenerator.EXPECT().
		GenerateCode().
		Return(expectedCode).
		Once()

	// Код уникален
	mockRepo.EXPECT().
		IsCodeUnique(expectedCode).
		Return(true).
		Once()

	// Создание или получение URL - создается новая запись
	mockRepo.EXPECT().
		CreateOrGetURL(expectedCode, model.URL("https://example.com"), "test-user").
		Return(expectedCode, true, nil). // true = создана новая запись
		Once()

	// Создаем service
	cfg := config.NewDefaultConfig()
	service := NewURLService(mockRepo, cfg)
	// Заменяем генератор на mock для теста
	service.codeGenerator = mockGenerator

	originalURL := model.URL("https://example.com")

	// Act
	code, created, err := service.CreateShortURL(originalURL, "test-user")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expectedCode, code)
	assert.True(t, created) // должна быть создана новая запись
}

// TestCreateShortURL_URLAlreadyExists проверяет возврат существующего кода при дублировании URL
func TestCreateShortURL_URLAlreadyExists(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	mockGenerator := mocks.NewMockGenerator(t)
	existingCode := model.Code("existing")
	newCode := model.Code("newcode")

	// Генератор возвращает код
	mockGenerator.EXPECT().
		GenerateCode().
		Return(newCode).
		Once()

	// Код уникален
	mockRepo.EXPECT().
		IsCodeUnique(newCode).
		Return(true).
		Once()

	// URL уже существует - возвращается существующий код
	mockRepo.EXPECT().
		CreateOrGetURL(newCode, model.URL("https://example.com"), "test-user").
		Return(existingCode, false, nil). // false = запись уже существовала
		Once()

	// Создаем service
	cfg := config.NewDefaultConfig()
	service := NewURLService(mockRepo, cfg)
	// Заменяем генератор на mock для теста
	service.codeGenerator = mockGenerator

	originalURL := model.URL("https://example.com")

	// Act
	code, created, err := service.CreateShortURL(originalURL, "test-user")

	// Assert
	require.NoError(t, err) // теперь ошибки не должно быть
	assert.Equal(t, existingCode, code)
	assert.False(t, created) // запись уже существовала
}
