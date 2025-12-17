package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStore_NewFileStore(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_urls.json")

	fs, err := NewFileStore(filePath)
	require.NoError(t, err)
	require.NotNil(t, fs)

	// Проверяем, что файл не создаётся, если нет данных
	_, err = os.Stat(filePath)
	assert.True(t, os.IsNotExist(err), "File should not exist when FileStore is created without data")
}

func TestFileStore_WriteAndRead(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_urls.json")

	fs, err := NewFileStore(filePath)
	require.NoError(t, err)

	// Записываем данные
	code := model.Code("abc123")
	url := model.URL("https://example.com")

	err = fs.Write(code, url)
	require.NoError(t, err)

	// Читаем данные
	result, err := fs.Read(code)
	require.NoError(t, err)
	assert.Equal(t, url, result)

	// Проверяем, что файл создан
	_, err = os.Stat(filePath)
	assert.NoError(t, err, "File should exist after Write")
}

func TestFileStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_urls.json")

	// Создаём первый FileStore и записываем данные
	fs1, err := NewFileStore(filePath)
	require.NoError(t, err)

	testData := URLMap{
		"code1": "https://example.com/1",
		"code2": "https://example.com/2",
		"code3": "https://example.com/3",
	}

	for code, url := range testData {
		err = fs1.Write(code, url)
		require.NoError(t, err)
	}

	// Создаём второй FileStore и проверяем, что данные загружены
	fs2, err := NewFileStore(filePath)
	require.NoError(t, err)

	for code, expectedURL := range testData {
		result, err := fs2.Read(code)
		require.NoError(t, err)
		assert.Equal(t, expectedURL, result)
	}
}

func TestFileStore_WriteExistingKey(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_urls.json")

	fs, err := NewFileStore(filePath)
	require.NoError(t, err)

	code := model.Code("abc123")
	url1 := model.URL("https://example.com/1")
	url2 := model.URL("https://example.com/2")

	// Первая запись должна пройти успешно
	err = fs.Write(code, url1)
	require.NoError(t, err)

	// Вторая запись с тем же ключом должна вернуть ошибку
	err = fs.Write(code, url2)
	assert.ErrorIs(t, err, ErrCodeAlreadyExists)
}

func TestFileStore_ReadNonExistentKey(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_urls.json")

	fs, err := NewFileStore(filePath)
	require.NoError(t, err)

	// Попытка прочитать несуществующий ключ
	_, err = fs.Read(model.Code("nonexistent"))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestFileStore_LoadFromExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_urls.json")

	// Создаём файл с данными вручную в JSONL формате (каждая запись на отдельной строке)
	jsonData := `{"uuid":"550e8400-e29b-41d4-a716-446655440000","short_url":"abc123","original_url":"https://example.com"}
{"uuid":"550e8400-e29b-41d4-a716-446655440001","short_url":"def456","original_url":"https://google.com"}
`
	err := os.WriteFile(filePath, []byte(jsonData), 0644)
	require.NoError(t, err)

	// Загружаем FileStore
	fs, err := NewFileStore(filePath)
	require.NoError(t, err)

	// Проверяем, что данные загружены
	url1, err := fs.Read(model.Code("abc123"))
	require.NoError(t, err)
	assert.Equal(t, model.URL("https://example.com"), url1)

	url2, err := fs.Read(model.Code("def456"))
	require.NoError(t, err)
	assert.Equal(t, model.URL("https://google.com"), url2)
}

func TestFileStore_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_urls.json")

	// Создаём пустой файл
	err := os.WriteFile(filePath, []byte(""), 0644)
	require.NoError(t, err)

	// FileStore должен успешно инициализироваться
	fs, err := NewFileStore(filePath)
	require.NoError(t, err)
	require.NotNil(t, fs)

	// Попытка прочитать должна вернуть ErrNotFound
	_, err = fs.Read(model.Code("any"))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestFileStore_MultipleWrites(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_urls.json")

	fs, err := NewFileStore(filePath)
	require.NoError(t, err)

	// Множественные записи
	for i := 0; i < 10; i++ {
		code := model.Code(string(rune('a' + i)))
		url := model.URL("https://example.com/" + string(rune('a'+i)))
		err = fs.Write(code, url)
		require.NoError(t, err)
	}

	// Перезагружаем FileStore
	fs2, err := NewFileStore(filePath)
	require.NoError(t, err)

	// Проверяем все записи
	for i := 0; i < 10; i++ {
		code := model.Code(string(rune('a' + i)))
		expectedURL := model.URL("https://example.com/" + string(rune('a'+i)))
		result, err := fs2.Read(code)
		require.NoError(t, err)
		assert.Equal(t, expectedURL, result)
	}
}
