package service

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/avc-dev/url-shortener/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestCreateShortURL_Success проверяет успешное создание короткого URL с первой попытки
func TestCreateShortURL_Success(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	mockRepo.EXPECT().
		CreateURL(mock.AnythingOfType("model.Code"), mock.AnythingOfType("model.URL")).
		Return(nil).
		Once()

	service := NewURLService(mockRepo)
	originalURL := model.URL("https://example.com")

	// Act
	code, err := service.CreateShortURL(originalURL)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, code)
	assert.Equal(t, CodeLength, len(code))

	// Проверяем что код содержит только разрешенные символы
	for _, char := range code {
		assert.True(t, strings.ContainsRune(AllowedChars, char),
			"Code contains invalid character: %c", char)
	}
}

// TestCreateShortURL_SuccessAfterCollision проверяет успех после одной коллизии
func TestCreateShortURL_SuccessAfterCollision(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	attemptCount := 0

	mockRepo.EXPECT().
		CreateURL(mock.AnythingOfType("model.Code"), mock.AnythingOfType("model.URL")).
		RunAndReturn(func(code model.Code, url model.URL) error {
			attemptCount++
			if attemptCount == 1 {
				return store.ErrAlreadyExists
			}
			return nil
		}).
		Maybe()

	service := NewURLService(mockRepo)
	originalURL := model.URL("https://example.com")

	// Act
	code, err := service.CreateShortURL(originalURL)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, code)
	assert.Equal(t, CodeLength, len(code))
	assert.Equal(t, 2, attemptCount)
}

// TestCreateShortURL_SuccessAfterMultipleCollisions проверяет успех после нескольких коллизий
func TestCreateShortURL_SuccessAfterMultipleCollisions(t *testing.T) {
	tests := []struct {
		name             string
		failUntilAttempt int
	}{
		{
			name:             "Success on second attempt",
			failUntilAttempt: 2,
		},
		{
			name:             "Success on fifth attempt",
			failUntilAttempt: 5,
		},
		{
			name:             "Success on tenth attempt",
			failUntilAttempt: 10,
		},
		{
			name:             "Success on 50th attempt",
			failUntilAttempt: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := mocks.NewMockURLRepository(t)
			attemptCount := 0

			mockRepo.EXPECT().
				CreateURL(mock.AnythingOfType("model.Code"), mock.AnythingOfType("model.URL")).
				RunAndReturn(func(code model.Code, url model.URL) error {
					attemptCount++
					if attemptCount < tt.failUntilAttempt {
						return store.ErrAlreadyExists
					}
					return nil
				}).
				Maybe()

			service := NewURLService(mockRepo)
			originalURL := model.URL("https://example.com")

			// Act
			code, err := service.CreateShortURL(originalURL)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, code)
			assert.Equal(t, CodeLength, len(code))
			assert.Equal(t, tt.failUntilAttempt, attemptCount)
		})
	}
}

// TestCreateShortURL_MaxRetriesExceeded проверяет исчерпание попыток
func TestCreateShortURL_MaxRetriesExceeded(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	attemptCount := 0

	mockRepo.EXPECT().
		CreateURL(mock.AnythingOfType("model.Code"), mock.AnythingOfType("model.URL")).
		RunAndReturn(func(code model.Code, url model.URL) error {
			attemptCount++
			return store.ErrAlreadyExists // Всегда коллизия
		}).
		Maybe()

	service := NewURLService(mockRepo)
	originalURL := model.URL("https://example.com")

	// Act
	code, err := service.CreateShortURL(originalURL)

	// Assert
	require.Error(t, err)
	assert.Empty(t, code)
	assert.Equal(t, MaxTries, attemptCount)
	assert.ErrorIs(t, err, ErrMaxRetriesExceeded)
}

// TestCreateShortURL_SuccessOnLastAttempt проверяет успех на последней попытке
func TestCreateShortURL_SuccessOnLastAttempt(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	attemptCount := 0

	mockRepo.EXPECT().
		CreateURL(mock.AnythingOfType("model.Code"), mock.AnythingOfType("model.URL")).
		RunAndReturn(func(code model.Code, url model.URL) error {
			attemptCount++
			if attemptCount < MaxTries {
				return store.ErrAlreadyExists
			}
			return nil
		}).
		Maybe()

	service := NewURLService(mockRepo)
	originalURL := model.URL("https://example.com")

	// Act
	code, err := service.CreateShortURL(originalURL)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, code)
	assert.Equal(t, MaxTries, attemptCount)
}

// TestCreateShortURL_DatabaseError проверяет немедленный возврат при ошибке БД
func TestCreateShortURL_DatabaseError(t *testing.T) {
	// Arrange
	dbError := errors.New("database connection failed")
	mockRepo := mocks.NewMockURLRepository(t)
	attemptCount := 0

	mockRepo.EXPECT().
		CreateURL(mock.AnythingOfType("model.Code"), mock.AnythingOfType("model.URL")).
		RunAndReturn(func(code model.Code, url model.URL) error {
			attemptCount++
			return dbError
		}).
		Once()

	service := NewURLService(mockRepo)
	originalURL := model.URL("https://example.com")

	// Act
	code, err := service.CreateShortURL(originalURL)

	// Assert
	require.Error(t, err)
	assert.Empty(t, code)
	assert.Equal(t, 1, attemptCount, "Should stop immediately on non-collision error")
	assert.ErrorIs(t, err, dbError)
}

// TestCreateShortURL_CollisionThenDatabaseError проверяет обработку коллизии затем ошибки БД
func TestCreateShortURL_CollisionThenDatabaseError(t *testing.T) {
	// Arrange
	dbError := errors.New("database error")
	mockRepo := mocks.NewMockURLRepository(t)
	attemptCount := 0

	mockRepo.EXPECT().
		CreateURL(mock.AnythingOfType("model.Code"), mock.AnythingOfType("model.URL")).
		RunAndReturn(func(code model.Code, url model.URL) error {
			attemptCount++
			if attemptCount <= 3 {
				return store.ErrAlreadyExists
			}
			return dbError
		}).
		Maybe()

	service := NewURLService(mockRepo)
	originalURL := model.URL("https://example.com")

	// Act
	code, err := service.CreateShortURL(originalURL)

	// Assert
	require.Error(t, err)
	assert.Empty(t, code)
	assert.Equal(t, 4, attemptCount)
	assert.ErrorIs(t, err, dbError)
}

// TestCreateShortURL_CodeFormat проверяет формат сгенерированных кодов
func TestCreateShortURL_CodeFormat(t *testing.T) {
	// Arrange
	numTests := 50
	var generatedCodes []model.Code

	mockRepo := mocks.NewMockURLRepository(t)
	mockRepo.EXPECT().
		CreateURL(mock.AnythingOfType("model.Code"), mock.AnythingOfType("model.URL")).
		RunAndReturn(func(code model.Code, url model.URL) error {
			generatedCodes = append(generatedCodes, code)
			return nil
		}).
		Times(numTests)

	service := NewURLService(mockRepo)
	originalURL := model.URL("https://example.com")

	// Act & Assert
	for i := 0; i < numTests; i++ {
		code, err := service.CreateShortURL(originalURL)
		require.NoError(t, err)

		// Проверяем длину
		assert.Equal(t, CodeLength, len(code), "Code: %s", code)

		// Проверяем что все символы из AllowedChars
		for _, char := range code {
			assert.True(t, strings.ContainsRune(AllowedChars, char),
				"Code %s contains invalid character: %c", code, char)
		}

		// Проверяем что нет пробелов
		assert.NotContains(t, string(code), " ", "Code %s contains spaces", code)

		// Проверяем что нет спецсимволов (только буквы)
		for _, char := range code {
			assert.True(t, (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z'),
				"Code %s contains non-letter character: %c", code, char)
		}
	}
}

// TestCreateShortURL_CodeUniqueness проверяет что генерируются разные коды
func TestCreateShortURL_CodeUniqueness(t *testing.T) {
	// Arrange
	numCodes := 100
	generatedCodes := make(map[model.Code]bool)

	mockRepo := mocks.NewMockURLRepository(t)
	mockRepo.EXPECT().
		CreateURL(mock.AnythingOfType("model.Code"), mock.AnythingOfType("model.URL")).
		RunAndReturn(func(code model.Code, url model.URL) error {
			generatedCodes[code] = true
			return nil
		}).
		Times(numCodes)

	service := NewURLService(mockRepo)
	originalURL := model.URL("https://example.com")

	// Act - генерируем множество кодов
	for i := 0; i < numCodes; i++ {
		_, err := service.CreateShortURL(originalURL)
		require.NoError(t, err)
	}

	// Assert - проверяем что большинство кодов уникальны
	uniqueCount := len(generatedCodes)
	minExpectedUnique := int(float64(numCodes) * 0.95)

	assert.GreaterOrEqual(t, uniqueCount, minExpectedUnique,
		"Expected at least %d unique codes out of %d", minExpectedUnique, numCodes)
}

// TestCreateShortURL_CodeCharacterDistribution проверяет распределение символов
func TestCreateShortURL_CodeCharacterDistribution(t *testing.T) {
	// Arrange
	numCodes := 1000
	charCount := make(map[rune]int)

	mockRepo := mocks.NewMockURLRepository(t)
	mockRepo.EXPECT().
		CreateURL(mock.AnythingOfType("model.Code"), mock.AnythingOfType("model.URL")).
		Return(nil).
		Times(numCodes)

	service := NewURLService(mockRepo)
	originalURL := model.URL("https://example.com")

	// Act - генерируем много кодов и считаем символы
	for i := 0; i < numCodes; i++ {
		code, err := service.CreateShortURL(originalURL)
		require.NoError(t, err)

		for _, char := range code {
			charCount[char]++
		}
	}

	// Assert - проверяем что используются разные символы
	uniqueCharsUsed := len(charCount)
	minExpectedChars := len(AllowedChars) / 2 // хотя бы половина

	assert.GreaterOrEqual(t, uniqueCharsUsed, minExpectedChars,
		"Expected at least %d different characters to be used", minExpectedChars)

	// Проверяем что все использованные символы из допустимого набора
	for char := range charCount {
		assert.True(t, strings.ContainsRune(AllowedChars, char),
			"Found invalid character in generated codes: %c", char)
	}
}

// TestCreateShortURL_Constants проверяет константы
func TestCreateShortURL_Constants(t *testing.T) {
	// Проверяем что константы имеют разумные значения
	assert.Greater(t, CodeLength, 0, "CodeLength should be positive")
	assert.Greater(t, MaxTries, 0, "MaxTries should be positive")
	assert.NotEmpty(t, AllowedChars, "AllowedChars should not be empty")

	// Проверяем ожидаемые значения
	assert.Equal(t, 8, CodeLength)
	assert.Equal(t, 100, MaxTries)
	assert.Equal(t, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", AllowedChars)
}

// TestCreateShortURL_ConcurrentGeneration проверяет параллельное создание
func TestCreateShortURL_ConcurrentGeneration(t *testing.T) {
	// Arrange
	numGoroutines := 50
	results := make(chan model.Code, numGoroutines)
	errorsChan := make(chan error, numGoroutines)
	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)

	mockRepo := mocks.NewMockURLRepository(t)
	mockRepo.EXPECT().
		CreateURL(mock.AnythingOfType("model.Code"), mock.AnythingOfType("model.URL")).
		Return(nil).
		Times(numGoroutines)

	service := NewURLService(mockRepo)
	originalURL := model.URL("https://example.com")

	// Act - запускаем параллельное создание
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			code, err := service.CreateShortURL(originalURL)
			if err != nil {
				errorsChan <- err
			} else {
				results <- code
			}
		}()
	}

	wg.Wait()
	close(results)
	close(errorsChan)

	// Assert - собираем результаты
	generatedCodes := make(map[model.Code]bool)
	for code := range results {
		assert.Equal(t, CodeLength, len(code))
		generatedCodes[code] = true
	}

	for err := range errorsChan {
		t.Errorf("Got error during concurrent generation: %v", err)
	}

	// Проверяем что большинство кодов уникальны
	uniqueCount := len(generatedCodes)
	minExpectedUnique := int(float64(numGoroutines) * 0.95)

	assert.GreaterOrEqual(t, uniqueCount, minExpectedUnique,
		"Expected at least %d unique codes out of %d", minExpectedUnique, numGoroutines)
}

// TestCreateShortURL_DifferentURLs проверяет создание для разных URL
func TestCreateShortURL_DifferentURLs(t *testing.T) {
	tests := []struct {
		name string
		url  model.URL
	}{
		{
			name: "Simple URL",
			url:  model.URL("https://example.com"),
		},
		{
			name: "URL with path",
			url:  model.URL("https://example.com/path/to/page"),
		},
		{
			name: "URL with query",
			url:  model.URL("https://example.com?query=param"),
		},
		{
			name: "Long URL",
			url:  model.URL("https://example.com/very/long/path/with/many/segments/and/parameters?foo=bar&baz=qux"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockRepo := mocks.NewMockURLRepository(t)
			var savedURL model.URL

			mockRepo.EXPECT().
				CreateURL(mock.AnythingOfType("model.Code"), mock.AnythingOfType("model.URL")).
				RunAndReturn(func(code model.Code, url model.URL) error {
					savedURL = url
					return nil
				}).
				Once()

			service := NewURLService(mockRepo)

			// Act
			code, err := service.CreateShortURL(tt.url)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, code)
			assert.Equal(t, tt.url, savedURL)
		})
	}
}
