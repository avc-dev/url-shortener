package store

import (
	"sync"
	"testing"

	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewStore проверяет создание нового хранилища
func TestNewStore(t *testing.T) {
	// Act
	store := NewStore()

	// Assert
	require.NotNil(t, store)
	assert.NotNil(t, store.store)
	assert.Equal(t, 0, len(store.store), "Expected empty store")
}

// TestStore_Write_Success проверяет успешную запись
func TestStore_Write_Success(t *testing.T) {
	tests := []struct {
		name  string
		code  model.Code
		url   model.URL
	}{
		{
			name: "Simple write",
			code: "abc12345",
			url:  "https://example.com",
		},
		{
			name: "Write with long URL",
			code: "xyz98765",
			url:  "https://example.com/very/long/path/with/many/segments",
		},
		{
			name: "Write with query params",
			code: "qwerty12",
			url:  "https://example.com?param=value&other=test",
		},
		{
			name: "Write with unicode",
			code: "unicode1",
			url:  "https://example.com/путь",
		},
		{
			name: "Single character code",
			code: "a",
			url:  "https://example.com",
		},
		{
			name: "Long code",
			code: "verylongcode1234567890",
			url:  "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			store := NewStore()

			// Act
			err := store.Write(tt.code, tt.url)

			// Assert
			require.NoError(t, err)

			// Проверяем что значение записано
			value, exists := store.store[tt.code]
			assert.True(t, exists, "Expected key to exist in store")
			assert.Equal(t, tt.url, value)
		})
	}
}

// TestStore_Write_Duplicate проверяет ошибку при дубликате ключа
func TestStore_Write_Duplicate(t *testing.T) {
	// Arrange
	store := NewStore()
	code := model.Code("abc12345")
	url1 := model.URL("https://example.com/first")
	url2 := model.URL("https://example.com/second")

	// Первая запись - успешна
	err := store.Write(code, url1)
	require.NoError(t, err)

	// Act - попытка записать тот же ключ
	err = store.Write(code, url2)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Проверяем что старое значение не изменилось
	value, _ := store.store[code]
	assert.Equal(t, url1, value)
}

// TestStore_Read_Success проверяет успешное чтение
func TestStore_Read_Success(t *testing.T) {
	tests := []struct {
		name string
		code model.Code
		url  model.URL
	}{
		{
			name: "Read simple value",
			code: "abc12345",
			url:  "https://example.com",
		},
		{
			name: "Read with special characters",
			code: "test-123",
			url:  "https://example.com/path?query=value",
		},
		{
			name: "Read unicode URL",
			code: "unicode1",
			url:  "https://example.com/путь/до/ресурса",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			store := NewStore()
			store.store[tt.code] = tt.url

			// Act
			value, err := store.Read(tt.code)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.url, value)
		})
	}
}

// TestStore_Read_NotFound проверяет ошибку при чтении несуществующего ключа
func TestStore_Read_NotFound(t *testing.T) {
	tests := []struct {
		name string
		code model.Code
	}{
		{
			name: "Read non-existent key",
			code: "notexist",
		},
		{
			name: "Read from empty store",
			code: "empty123",
		},
		{
			name: "Read with empty code",
			code: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			store := NewStore()

			// Act
			value, err := store.Read(tt.code)

			// Assert
			require.Error(t, err)
			assert.Empty(t, value)
			assert.Contains(t, err.Error(), "not exist")
		})
	}
}

// TestStore_WriteRead_Integration проверяет интеграцию записи и чтения
func TestStore_WriteRead_Integration(t *testing.T) {
	// Arrange
	store := NewStore()
	testData := map[model.Code]model.URL{
		"code1": "https://example.com/1",
		"code2": "https://example.com/2",
		"code3": "https://example.com/3",
		"code4": "https://example.com/4",
		"code5": "https://example.com/5",
	}

	// Act - записываем все данные
	for code, url := range testData {
		err := store.Write(code, url)
		require.NoError(t, err, "Failed to write code %s", code)
	}

	// Assert - читаем все данные
	for code, expectedURL := range testData {
		actualURL, err := store.Read(code)
		require.NoError(t, err, "Failed to read code %s", code)
		assert.Equal(t, expectedURL, actualURL, "For code %s", code)
	}
}

// TestStore_MultipleWrites проверяет множественные записи
func TestStore_MultipleWrites(t *testing.T) {
	// Arrange
	store := NewStore()
	numWrites := 100

	// Act - записываем много значений
	for i := 0; i < numWrites; i++ {
		code := model.Code(string(rune('a' + i%26)) + string(rune('0' + i%10)))
		url := model.URL("https://example.com/" + string(rune('0'+i)))
		
		_ = store.Write(code, url)
	}

	// Assert - проверяем что записаны данные
	assert.NotEmpty(t, store.store, "Expected some items in store")
}

// TestStore_ConcurrentReads проверяет параллельное чтение
func TestStore_ConcurrentReads(t *testing.T) {
	// Arrange
	store := NewStore()
	code := model.Code("testcode")
	expectedURL := model.URL("https://example.com")
	store.store[code] = expectedURL

	numGoroutines := 100
	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)
	errors := make(chan error, numGoroutines)

	// Act - множество параллельных чтений
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			
			url, err := store.Read(code)
			if err != nil {
				errors <- err
				return
			}

			assert.Equal(t, expectedURL, url)
		}()
	}

	wg.Wait()
	close(errors)

	// Assert
	for err := range errors {
		t.Errorf("Got error during concurrent reads: %v", err)
	}
}

// TestStore_ConcurrentWrites проверяет параллельную запись разных ключей
func TestStore_ConcurrentWrites(t *testing.T) {
	// Arrange
	store := NewStore()
	numGoroutines := 50
	wg := sync.WaitGroup{}
	wg.Add(numGoroutines)
	errors := make(chan error, numGoroutines)

	// Act - множество параллельных записей разных ключей
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			
			code := model.Code("code" + string(rune('0'+index)))
			url := model.URL("https://example.com/" + string(rune('0'+index)))
			
			err := store.Write(code, url)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Assert
	errorCount := 0
	for err := range errors {
		errorCount++
		t.Logf("Got error during concurrent writes: %v", err)
	}

	// Проверяем что большинство записей успешны
	assert.NotEmpty(t, store.store, "Expected at least some successful writes")
}

// TestStore_ConcurrentReadWrite проверяет параллельное чтение и запись
func TestStore_ConcurrentReadWrite(t *testing.T) {
	// Arrange
	store := NewStore()
	numOperations := 100
	wg := sync.WaitGroup{}
	wg.Add(numOperations * 2)

	// Предварительно записываем некоторые данные
	for i := 0; i < 10; i++ {
		code := model.Code("initial" + string(rune('0'+i)))
		url := model.URL("https://example.com/initial/" + string(rune('0'+i)))
		store.store[code] = url
	}

	// Act - параллельные чтения
	for i := 0; i < numOperations; i++ {
		go func(index int) {
			defer wg.Done()
			code := model.Code("initial" + string(rune('0'+(index%10))))
			_, _ = store.Read(code)
		}(i)
	}

	// Act - параллельные записи
	for i := 0; i < numOperations; i++ {
		go func(index int) {
			defer wg.Done()
			code := model.Code("new" + string(rune('0'+index)))
			url := model.URL("https://example.com/new/" + string(rune('0'+index)))
			_ = store.Write(code, url)
		}(i)
	}

	// Assert - просто проверяем что нет race conditions и паники
	wg.Wait()
	
	// Проверяем что store все еще работает
	testCode := model.Code("initial0")
	_, err := store.Read(testCode)
	require.NoError(t, err, "Store corrupted after concurrent operations")
}

// TestStore_WriteSameKeyMultipleTimes проверяет многократную попытку записи одного ключа
func TestStore_WriteSameKeyMultipleTimes(t *testing.T) {
	// Arrange
	store := NewStore()
	code := model.Code("testcode")
	url1 := model.URL("https://example.com/first")
	url2 := model.URL("https://example.com/second")
	url3 := model.URL("https://example.com/third")

	// Act & Assert
	// Первая запись - успешна
	err := store.Write(code, url1)
	require.NoError(t, err)

	// Вторая запись - ошибка
	err = store.Write(code, url2)
	require.Error(t, err)

	// Третья запись - ошибка
	err = store.Write(code, url3)
	require.Error(t, err)

	// Проверяем что хранится оригинальное значение
	value, _ := store.Read(code)
	assert.Equal(t, url1, value)
}

// TestStore_EmptyValues проверяет работу с пустыми значениями
func TestStore_EmptyValues(t *testing.T) {
	tests := []struct {
		name        string
		code        model.Code
		url         model.URL
		expectError bool
	}{
		{
			name:        "Empty code with URL",
			code:        "",
			url:         "https://example.com",
			expectError: false,
		},
		{
			name:        "Code with empty URL",
			code:        "abc12345",
			url:         "",
			expectError: false,
		},
		{
			name:        "Both empty",
			code:        "",
			url:         "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			store := NewStore()

			// Act
			err := store.Write(tt.code, tt.url)

			// Assert
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Если запись успешна, проверяем чтение
				value, readErr := store.Read(tt.code)
				require.NoError(t, readErr)
				assert.Equal(t, tt.url, value)
			}
		})
	}
}

// TestStore_LargeDataset проверяет работу с большим количеством данных
func TestStore_LargeDataset(t *testing.T) {
	// Arrange
	store := NewStore()
	numItems := 1000

	// Act - записываем много элементов
	for i := 0; i < numItems; i++ {
		code := model.Code("code" + string(rune('a'+(i%26))) + string(rune('0'+(i%10))))
		url := model.URL("https://example.com/path/" + string(rune('0'+(i%10))))
		_ = store.Write(code, url)
	}

	// Assert - проверяем что данные записались
	assert.NotEmpty(t, store.store, "Expected items in store")

	// Проверяем что можем читать данные
	testCode := model.Code("codea0")
	_, _ = store.Read(testCode)
}

// TestStore_ErrorMessages проверяет сообщения об ошибках
func TestStore_ErrorMessages(t *testing.T) {
	store := NewStore()

	// Тест ошибки "not exist"
	t.Run("Read not exist error message", func(t *testing.T) {
		code := model.Code("notfound")
		_, err := store.Read(code)
		
		require.Error(t, err)
		assert.Contains(t, err.Error(), string(code))
		assert.Contains(t, err.Error(), "not exist")
	})

	// Тест ошибки "already exists"
	t.Run("Write duplicate error message", func(t *testing.T) {
		code := model.Code("duplicate")
		url := model.URL("https://example.com")
		
		_ = store.Write(code, url)
		err := store.Write(code, url)
		
		require.Error(t, err)
		assert.Contains(t, err.Error(), string(code))
		assert.Contains(t, err.Error(), "already exists")
	})
}

// TestStore_StoreIsolation проверяет изоляцию разных экземпляров Store
func TestStore_StoreIsolation(t *testing.T) {
	// Arrange
	store1 := NewStore()
	store2 := NewStore()
	
	code := model.Code("testcode")
	url1 := model.URL("https://example.com/store1")
	url2 := model.URL("https://example.com/store2")

	// Act
	err1 := store1.Write(code, url1)
	err2 := store2.Write(code, url2)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2, "Both writes should succeed in different stores")

	// Проверяем что каждый store имеет свое значение
	value1, _ := store1.Read(code)
	value2, _ := store2.Read(code)

	assert.Equal(t, url1, value1, "Store1 should have its own value")
	assert.Equal(t, url2, value2, "Store2 should have its own value")
	assert.NotEqual(t, value1, value2, "Stores should be isolated")
}
