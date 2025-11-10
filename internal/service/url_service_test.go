package service

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestGenerateUniqueCode_Success проверяет успешную генерацию кода
func TestGenerateUniqueCode_Success(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "First attempt success",
		},
		{
			name: "Generate multiple times",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockChecker := mocks.NewMockCodeChecker(t)
			mockChecker.EXPECT().
				Exists(mock.AnythingOfType("model.Code")).
				Return(false, nil)

			service := NewURLService(mockChecker)

			// Act
			code, err := service.GenerateUniqueCode()

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, code)
			assert.Equal(t, CodeLength, len(code))

			// Проверяем что код содержит только разрешенные символы
			for _, char := range code {
				assert.True(t, strings.ContainsRune(AllowedChars, char),
					"Code contains invalid character: %c", char)
			}
		})
	}
}

// TestGenerateUniqueCode_SuccessAfterRetries проверяет успех после нескольких попыток
func TestGenerateUniqueCode_SuccessAfterRetries(t *testing.T) {
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
			mockChecker := mocks.NewMockCodeChecker(t)
			attemptCount := 0

			mockChecker.EXPECT().
				Exists(mock.AnythingOfType("model.Code")).
				RunAndReturn(func(code model.Code) (bool, error) {
					attemptCount++
					return attemptCount < tt.failUntilAttempt, nil // true = существует
				}).
				Maybe() // Может вызваться много раз

			service := NewURLService(mockChecker)

			// Act
			code, err := service.GenerateUniqueCode()

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, code)
			assert.Equal(t, CodeLength, len(code))
			assert.GreaterOrEqual(t, attemptCount, tt.failUntilAttempt)
		})
	}
}

// TestGenerateUniqueCode_MaxTriesExceeded проверяет исчерпание попыток
func TestGenerateUniqueCode_MaxTriesExceeded(t *testing.T) {
	// Arrange
	mockChecker := mocks.NewMockCodeChecker(t)
	attemptCount := 0

	mockChecker.EXPECT().
		Exists(mock.AnythingOfType("model.Code")).
		RunAndReturn(func(code model.Code) (bool, error) {
			attemptCount++
			return true, nil // Всегда возвращаем что код существует
		}).
		Maybe()

	service := NewURLService(mockChecker)

	// Act
	code, err := service.GenerateUniqueCode()

	// Assert
	require.Error(t, err)
	assert.Empty(t, code)
	assert.Equal(t, MaxTries, attemptCount)
	assert.Contains(t, err.Error(), "could not generate unique code")
}

// TestGenerateUniqueCode_MaxTriesExactly проверяет успех на последней попытке
func TestGenerateUniqueCode_MaxTriesExactly(t *testing.T) {
	// Arrange
	mockChecker := mocks.NewMockCodeChecker(t)
	attemptCount := 0

	mockChecker.EXPECT().
		Exists(mock.AnythingOfType("model.Code")).
		RunAndReturn(func(code model.Code) (bool, error) {
			attemptCount++
			return attemptCount < MaxTries, nil // Успех только на последней попытке
		}).
		Maybe()

	service := NewURLService(mockChecker)

	// Act
	code, err := service.GenerateUniqueCode()

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, code)
	assert.Equal(t, MaxTries, attemptCount)
}

// TestGenerateUniqueCode_CodeUniqueness проверяет что генерируются разные коды
func TestGenerateUniqueCode_CodeUniqueness(t *testing.T) {
	// Arrange
	numCodes := 100
	generatedCodes := make(map[model.Code]bool)

	mockChecker := mocks.NewMockCodeChecker(t)
	mockChecker.EXPECT().
		Exists(mock.AnythingOfType("model.Code")).
		Return(false, nil).
		Maybe()

	service := NewURLService(mockChecker)

	// Act - генерируем множество кодов
	for i := 0; i < numCodes; i++ {
		code, err := service.GenerateUniqueCode()
		require.NoError(t, err)
		generatedCodes[code] = true
	}

	// Assert - проверяем что большинство кодов уникальны
	uniqueCount := len(generatedCodes)
	minExpectedUnique := int(float64(numCodes) * 0.95)

	assert.GreaterOrEqual(t, uniqueCount, minExpectedUnique,
		"Expected at least %d unique codes out of %d", minExpectedUnique, numCodes)
}

// TestGenerateUniqueCode_CodeFormat проверяет формат сгенерированного кода
func TestGenerateUniqueCode_CodeFormat(t *testing.T) {
	// Arrange
	numTests := 50

	mockChecker := mocks.NewMockCodeChecker(t)
	mockChecker.EXPECT().
		Exists(mock.AnythingOfType("model.Code")).
		Return(false, nil).
		Maybe()

	service := NewURLService(mockChecker)

	// Act & Assert
	for i := 0; i < numTests; i++ {
		code, err := service.GenerateUniqueCode()
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

// TestGenerateUniqueCode_CheckerError проверяет обработку ошибок от checker
func TestGenerateUniqueCode_CheckerError(t *testing.T) {
	tests := []struct {
		name          string
		checkerError  error
		failUntil     int
		expectSuccess bool
	}{
		{
			name:          "Database error then success",
			checkerError:  errors.New("database connection failed"),
			failUntil:     3,
			expectSuccess: true,
		},
		{
			name:          "Temporary error then success",
			checkerError:  errors.New("temporary error"),
			failUntil:     5,
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockChecker := mocks.NewMockCodeChecker(t)
			attemptCount := 0

			mockChecker.EXPECT().
				Exists(mock.AnythingOfType("model.Code")).
				RunAndReturn(func(code model.Code) (bool, error) {
					attemptCount++
					if attemptCount <= tt.failUntil {
						return false, tt.checkerError
					}
					return false, nil
				}).
				Maybe()

			service := NewURLService(mockChecker)

			// Act
			code, err := service.GenerateUniqueCode()

			// Assert
			if tt.expectSuccess {
				require.NoError(t, err)
				assert.NotEmpty(t, code)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestGenerateUniqueCode_CodeCharacterDistribution проверяет распределение символов
func TestGenerateUniqueCode_CodeCharacterDistribution(t *testing.T) {
	// Arrange
	numCodes := 1000
	charCount := make(map[rune]int)

	mockChecker := mocks.NewMockCodeChecker(t)
	mockChecker.EXPECT().
		Exists(mock.AnythingOfType("model.Code")).
		Return(false, nil).
		Maybe()

	service := NewURLService(mockChecker)

	// Act - генерируем много кодов и считаем символы
	for i := 0; i < numCodes; i++ {
		code, err := service.GenerateUniqueCode()
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

// TestGenerateUniqueCode_Constants проверяет константы
func TestGenerateUniqueCode_Constants(t *testing.T) {
	// Проверяем что константы имеют разумные значения
	assert.Greater(t, CodeLength, 0, "CodeLength should be positive")
	assert.Greater(t, MaxTries, 0, "MaxTries should be positive")
	assert.NotEmpty(t, AllowedChars, "AllowedChars should not be empty")

	// Проверяем ожидаемые значения
	assert.Equal(t, 8, CodeLength)
	assert.Equal(t, 100, MaxTries)
	assert.Equal(t, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", AllowedChars)
}

// TestGenerateUniqueCode_ConcurrentGeneration проверяет параллельную генерацию
func TestGenerateUniqueCode_ConcurrentGeneration(t *testing.T) {
	// Arrange
	numGoroutines := 50
	results := make(chan model.Code, numGoroutines)
	errors := make(chan error, numGoroutines)
	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)

	mockChecker := mocks.NewMockCodeChecker(t)
	mockChecker.EXPECT().
		Exists(mock.AnythingOfType("model.Code")).
		Return(false, nil).
		Maybe()

	service := NewURLService(mockChecker)

	// Act - запускаем параллельную генерацию
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			code, err := service.GenerateUniqueCode()
			if err != nil {
				errors <- err
			} else {
				results <- code
			}
		}()
	}

	wg.Wait()
	close(results)
	close(errors)

	// Assert - собираем результаты
	generatedCodes := make(map[model.Code]bool)
	for code := range results {
		assert.Equal(t, CodeLength, len(code))
		generatedCodes[code] = true
	}

	for err := range errors {
		t.Errorf("Got error during concurrent generation: %v", err)
	}

	// Проверяем что большинство кодов уникальны
	uniqueCount := len(generatedCodes)
	minExpectedUnique := int(float64(numGoroutines) * 0.95)

	assert.GreaterOrEqual(t, uniqueCount, minExpectedUnique,
		"Expected at least %d unique codes out of %d", minExpectedUnique, numGoroutines)
}

// TestCreateShortURL_Success проверяет успешное создание короткого URL
func TestCreateShortURL_Success(t *testing.T) {
	// Arrange
	mockChecker := mocks.NewMockCodeChecker(t)
	mockChecker.EXPECT().
		Exists(mock.AnythingOfType("model.Code")).
		Return(false, nil)

	service := NewURLService(mockChecker)
	originalURL := model.URL("https://example.com")

	// Act
	code, err := service.CreateShortURL(originalURL)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, code)
	assert.Equal(t, CodeLength, len(code))
}

// TestCreateShortURL_GenerationFails проверяет ошибку при генерации кода
func TestCreateShortURL_GenerationFails(t *testing.T) {
	// Arrange
	mockChecker := mocks.NewMockCodeChecker(t)
	// Все коды заняты
	mockChecker.EXPECT().
		Exists(mock.AnythingOfType("model.Code")).
		Return(true, nil).
		Maybe()

	service := NewURLService(mockChecker)
	originalURL := model.URL("https://example.com")

	// Act
	code, err := service.CreateShortURL(originalURL)

	// Assert
	require.Error(t, err)
	assert.Empty(t, code)
	assert.Contains(t, err.Error(), "failed to generate unique code")
}
