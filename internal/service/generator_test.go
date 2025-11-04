package service

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateCode_Success проверяет успешную генерацию кода
func TestGenerateCode_Success(t *testing.T) {
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
			checkerFunc := func(code string) error {
				// Всегда возвращаем успех
				return nil
			}

			// Act
			code, err := GenerateCode(checkerFunc)

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

// TestGenerateCode_SuccessAfterRetries проверяет успех после нескольких попыток
func TestGenerateCode_SuccessAfterRetries(t *testing.T) {
	tests := []struct {
		name              string
		failUntilAttempt  int
		expectedAttempts  int
	}{
		{
			name:             "Success on second attempt",
			failUntilAttempt: 2,
			expectedAttempts: 2,
		},
		{
			name:             "Success on fifth attempt",
			failUntilAttempt: 5,
			expectedAttempts: 5,
		},
		{
			name:             "Success on tenth attempt",
			failUntilAttempt: 10,
			expectedAttempts: 10,
		},
		{
			name:             "Success on 50th attempt",
			failUntilAttempt: 50,
			expectedAttempts: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			attemptCount := 0
			checkerFunc := func(code string) error {
				attemptCount++
				if attemptCount < tt.failUntilAttempt {
					return errors.New("code already exists")
				}
				return nil
			}

			// Act
			code, err := GenerateCode(checkerFunc)

			// Assert
			require.NoError(t, err)
			assert.NotEmpty(t, code)
			assert.Equal(t, tt.expectedAttempts, attemptCount)
			assert.Equal(t, CodeLength, len(code))
		})
	}
}

// TestGenerateCode_MaxTriesExceeded проверяет исчерпание попыток
func TestGenerateCode_MaxTriesExceeded(t *testing.T) {
	// Arrange
	attemptCount := 0
	checkerFunc := func(code string) error {
		attemptCount++
		// Всегда возвращаем ошибку
		return errors.New("duplicate code")
	}

	// Act
	code, err := GenerateCode(checkerFunc)

	// Assert
	require.Error(t, err)
	assert.Empty(t, code)
	assert.Equal(t, MaxTries, attemptCount)
	assert.Equal(t, "could not generate unique code", err.Error())
}

// TestGenerateCode_MaxTriesExactly проверяет успех на последней попытке
func TestGenerateCode_MaxTriesExactly(t *testing.T) {
	// Arrange
	attemptCount := 0
	checkerFunc := func(code string) error {
		attemptCount++
		// Успех только на последней (MaxTries) попытке
		if attemptCount < MaxTries {
			return errors.New("duplicate code")
		}
		return nil
	}

	// Act
	code, err := GenerateCode(checkerFunc)

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, code)
	assert.Equal(t, MaxTries, attemptCount)
}

// TestGenerateCode_CodeUniqueness проверяет что генерируются разные коды
func TestGenerateCode_CodeUniqueness(t *testing.T) {
	// Arrange
	numCodes := 100
	generatedCodes := make(map[string]bool)
	checkerFunc := func(code string) error {
		return nil
	}

	// Act - генерируем множество кодов
	for i := 0; i < numCodes; i++ {
		code, err := GenerateCode(checkerFunc)
		require.NoError(t, err)
		generatedCodes[code] = true
	}

	// Assert - проверяем что большинство кодов уникальны
	uniqueCount := len(generatedCodes)
	minExpectedUnique := int(float64(numCodes) * 0.95)
	
	assert.GreaterOrEqual(t, uniqueCount, minExpectedUnique,
		"Expected at least %d unique codes out of %d", minExpectedUnique, numCodes)
}

// TestGenerateCode_CodeFormat проверяет формат сгенерированного кода
func TestGenerateCode_CodeFormat(t *testing.T) {
	// Arrange
	numTests := 50
	checkerFunc := func(code string) error {
		return nil
	}

	// Act & Assert
	for i := 0; i < numTests; i++ {
		code, err := GenerateCode(checkerFunc)
		require.NoError(t, err)

		// Проверяем длину
		assert.Equal(t, CodeLength, len(code), "Code: %s", code)

		// Проверяем что все символы из AllowedChars
		for _, char := range code {
			assert.True(t, strings.ContainsRune(AllowedChars, char),
				"Code %s contains invalid character: %c", code, char)
		}

		// Проверяем что нет пробелов
		assert.NotContains(t, code, " ", "Code %s contains spaces", code)

		// Проверяем что нет спецсимволов (только буквы)
		for _, char := range code {
			assert.True(t, (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z'),
				"Code %s contains non-letter character: %c", code, char)
		}
	}
}

// TestGenerateCode_CheckerFunctionCalled проверяет что checker функция вызывается
func TestGenerateCode_CheckerFunctionCalled(t *testing.T) {
	// Arrange
	called := false
	var capturedCode string
	
	checkerFunc := func(code string) error {
		called = true
		capturedCode = code
		return nil
	}

	// Act
	code, err := GenerateCode(checkerFunc)

	// Assert
	require.NoError(t, err)
	assert.True(t, called, "Expected checker function to be called")
	assert.Equal(t, code, capturedCode)
}

// TestGenerateCode_CheckerFunctionReceivesDifferentCodes проверяет разные коды в checker
func TestGenerateCode_CheckerFunctionReceivesDifferentCodes(t *testing.T) {
	// Arrange
	receivedCodes := make(map[string]bool)
	failCount := 10
	attemptCount := 0

	checkerFunc := func(code string) error {
		attemptCount++
		receivedCodes[code] = true
		
		// Первые failCount попыток возвращают ошибку
		if attemptCount <= failCount {
			return errors.New("duplicate")
		}
		return nil
	}

	// Act
	_, err := GenerateCode(checkerFunc)

	// Assert
	require.NoError(t, err)
	assert.Greater(t, len(receivedCodes), 1,
		"Expected checker to receive multiple different codes")
}

// TestGenerateCode_DifferentErrors проверяет обработку разных типов ошибок
func TestGenerateCode_DifferentErrors(t *testing.T) {
	tests := []struct {
		name          string
		checkerError  error
		failUntil     int
		expectSuccess bool
	}{
		{
			name:          "Duplicate code error then success",
			checkerError:  errors.New("duplicate code"),
			failUntil:     3,
			expectSuccess: true,
		},
		{
			name:          "Database error then success",
			checkerError:  errors.New("database connection failed"),
			failUntil:     5,
			expectSuccess: true,
		},
		{
			name:          "Generic error then success",
			checkerError:  errors.New("some error"),
			failUntil:     2,
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			attemptCount := 0
			checkerFunc := func(code string) error {
				attemptCount++
				if attemptCount <= tt.failUntil {
					return tt.checkerError
				}
				return nil
			}

			// Act
			code, err := GenerateCode(checkerFunc)

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

// TestGenerateCode_CodeCharacterDistribution проверяет распределение символов
func TestGenerateCode_CodeCharacterDistribution(t *testing.T) {
	// Arrange
	numCodes := 1000
	charCount := make(map[rune]int)
	checkerFunc := func(code string) error {
		return nil
	}

	// Act - генерируем много кодов и считаем символы
	for i := 0; i < numCodes; i++ {
		code, err := GenerateCode(checkerFunc)
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

// TestGenerateCode_Constants проверяет константы
func TestGenerateCode_Constants(t *testing.T) {
	// Проверяем что константы имеют разумные значения
	assert.Greater(t, CodeLength, 0, "CodeLength should be positive")
	assert.Greater(t, MaxTries, 0, "MaxTries should be positive")
	assert.NotEmpty(t, AllowedChars, "AllowedChars should not be empty")

	// Проверяем ожидаемые значения
	assert.Equal(t, 8, CodeLength)
	assert.Equal(t, 100, MaxTries)
	assert.Equal(t, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", AllowedChars)
}

// TestGenerateCode_NilChecker проверяет поведение с nil checker
func TestGenerateCode_NilChecker(t *testing.T) {
	// Arrange
	defer func() {
		r := recover()
		assert.NotNil(t, r, "Expected panic with nil checker function")
	}()

	// Act - это должно вызвать панику
	GenerateCode(nil)
}

// TestGenerateCode_ConcurrentGeneration проверяет параллельную генерацию
func TestGenerateCode_ConcurrentGeneration(t *testing.T) {
	// Arrange
	numGoroutines := 50
	results := make(chan string, numGoroutines)
	errors := make(chan error, numGoroutines)
	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)

	checkerFunc := func(code string) error {
		return nil
	}

	// Act - запускаем параллельную генерацию
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			code, err := GenerateCode(checkerFunc)
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
	generatedCodes := make(map[string]bool)
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
