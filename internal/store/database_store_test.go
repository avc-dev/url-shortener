package store

import (
	"context"
	"os"
	"testing"

	"github.com/avc-dev/url-shortener/internal/config/db"
	"github.com/avc-dev/url-shortener/internal/migrations"
	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupTestDB создает тестовую базу данных для интеграционных тестов
func setupTestDB(t *testing.T) (*DatabaseStore, func()) {
	t.Helper()

	// Получаем DSN из переменных окружения или используем значение по умолчанию
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5433/urlshortener?sslmode=disable"
	}

	// Создаем конфигурацию БД
	dbConfig := db.NewConfig(dsn)
	database, err := dbConfig.Connect(context.Background())
	require.NoError(t, err)

	// Запускаем миграции
	logger := zap.NewNop()
	migrator := migrations.NewMigrator(database.DB(), logger)
	err = migrator.RunUp()
	require.NoError(t, err)

	// Создаем database store
	store := NewDatabaseStore(database)

	// Очищаем таблицы перед каждым тестом через store
	// Получаем доступ к pool через type assertion (для тестов это допустимо)
	adapter, ok := database.(*db.DBAdapter)
	require.True(t, ok, "Expected DBAdapter")
	_, err = adapter.Pool.Exec(context.Background(), "DELETE FROM urls")
	require.NoError(t, err)

	// Возвращаем cleanup функцию
	cleanup := func() {
		database.Close()
	}

	return store, cleanup
}

// TestDatabaseStore_DeleteURLsBatch_Success проверяет успешное batch удаление URL
func TestDatabaseStore_DeleteURLsBatch_Success(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	userID := "test-user"
	codes := []model.Code{"code1", "code2", "code3"}
	urls := []model.URL{"https://example1.com", "https://example2.com", "https://example3.com"}

	// Создаем тестовые данные
	for i, code := range codes {
		_, _, err := store.CreateOrGetURL(code, urls[i], userID)
		require.NoError(t, err)
	}

	// Проверяем что URL созданы и не удалены
	for _, code := range codes {
		url, err := store.Read(code)
		require.NoError(t, err)
		assert.NotEmpty(t, url)

		// Проверяем что URL принадлежит пользователю
		assert.True(t, store.IsURLOwnedByUser(code, userID))
	}

	// Act - удаляем URL
	err := store.DeleteURLsBatch(codes, userID)

	// Assert
	assert.NoError(t, err)

	// Проверяем что URL помечены как удаленные
	for _, code := range codes {
		_, err := store.Read(code)
		assert.Error(t, err, "URL should be marked as deleted")

		// Удаленный URL не считается принадлежащим пользователю
		assert.False(t, store.IsURLOwnedByUser(code, userID), "Deleted URL should not be owned by user")
	}
}

// TestDatabaseStore_DeleteURLsBatch_EmptyCodes проверяет обработку пустого списка кодов
func TestDatabaseStore_DeleteURLsBatch_EmptyCodes(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	userID := "test-user"
	var codes []model.Code

	// Act
	err := store.DeleteURLsBatch(codes, userID)

	// Assert
	assert.NoError(t, err)
}

// TestDatabaseStore_DeleteURLsBatch_SingleCode проверяет удаление одного URL
func TestDatabaseStore_DeleteURLsBatch_SingleCode(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	userID := "test-user"
	codes := []model.Code{"single"}
	url := model.URL("https://example.com")

	// Создаем тестовые данные
	_, _, err := store.CreateOrGetURL(codes[0], url, userID)
	require.NoError(t, err)

	// Act - удаляем URL
	err = store.DeleteURLsBatch(codes, userID)

	// Assert
	assert.NoError(t, err)

	// Проверяем что URL помечен как удаленный
	_, err = store.Read(codes[0])
	assert.Error(t, err)
}

// TestDatabaseStore_DeleteURLsBatch_WrongUser проверяет что URL другого пользователя не удаляются
func TestDatabaseStore_DeleteURLsBatch_WrongUser(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	userID1 := "user1"
	userID2 := "user2"
	code := model.Code("shared")
	url := model.URL("https://example.com")

	// Создаем URL для user1
	_, _, err := store.CreateOrGetURL(code, url, userID1)
	require.NoError(t, err)

	// Act - пытаемся удалить URL от имени user2
	err = store.DeleteURLsBatch([]model.Code{code}, userID2)

	// Assert
	assert.NoError(t, err)

	// Проверяем что URL все еще доступен для user1
	retrievedURL, err := store.Read(code)
	assert.NoError(t, err)
	assert.Equal(t, url, retrievedURL)

	// Проверяем что URL принадлежит user1, но не user2
	assert.True(t, store.IsURLOwnedByUser(code, userID1))
	assert.False(t, store.IsURLOwnedByUser(code, userID2))
}

// TestDatabaseStore_IsURLOwnedByUser_Success проверяет успешную проверку принадлежности URL
func TestDatabaseStore_IsURLOwnedByUser_Success(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	userID := "test-user"
	code := model.Code("owned")
	url := model.URL("https://example.com")

	// Создаем URL
	_, _, err := store.CreateOrGetURL(code, url, userID)
	require.NoError(t, err)

	// Act & Assert
	assert.True(t, store.IsURLOwnedByUser(code, userID))
}

// TestDatabaseStore_IsURLOwnedByUser_WrongUser проверяет проверку принадлежности для другого пользователя
func TestDatabaseStore_IsURLOwnedByUser_WrongUser(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	userID1 := "user1"
	userID2 := "user2"
	code := model.Code("owned")
	url := model.URL("https://example.com")

	// Создаем URL для user1
	_, _, err := store.CreateOrGetURL(code, url, userID1)
	require.NoError(t, err)

	// Act & Assert
	assert.True(t, store.IsURLOwnedByUser(code, userID1))
	assert.False(t, store.IsURLOwnedByUser(code, userID2))
}

// TestDatabaseStore_IsURLOwnedByUser_DeletedURL проверяет что удаленный URL не принадлежит пользователю
func TestDatabaseStore_IsURLOwnedByUser_DeletedURL(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	userID := "test-user"
	code := model.Code("deleted")
	url := model.URL("https://example.com")

	// Создаем URL
	_, _, err := store.CreateOrGetURL(code, url, userID)
	require.NoError(t, err)

	// Удаляем URL
	err = store.DeleteURLsBatch([]model.Code{code}, userID)
	require.NoError(t, err)

	// Act & Assert - удаленный URL не принадлежит пользователю
	assert.False(t, store.IsURLOwnedByUser(code, userID))
}

// TestDatabaseStore_IsURLOwnedByUser_NonExistentURL проверяет проверку принадлежности для несуществующего URL
func TestDatabaseStore_IsURLOwnedByUser_NonExistentURL(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	userID := "test-user"
	code := model.Code("nonexistent")

	// Act & Assert
	assert.False(t, store.IsURLOwnedByUser(code, userID))
}

// TestDatabaseStore_batchUpdateDeletedFlag_Success проверяет успешное обновление флага is_deleted
func TestDatabaseStore_batchUpdateDeletedFlag_Success(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	userID := "test-user"
	codes := []model.Code{"batch1", "batch2"}
	urls := []model.URL{"https://batch1.com", "https://batch2.com"}

	// Создаем тестовые данные
	for i, code := range codes {
		_, _, err := store.CreateOrGetURL(code, urls[i], userID)
		require.NoError(t, err)
	}

	// Act - помечаем как удаленные
	err := store.batchUpdateDeletedFlag(context.Background(), codes, userID, true)

	// Assert
	assert.NoError(t, err)

	// Проверяем что URL помечены как удаленные
	for _, code := range codes {
		_, err := store.Read(code)
		assert.Error(t, err)
	}
}

// TestDatabaseStore_batchUpdateDeletedFlag_EmptyCodes проверяет обработку пустого списка кодов
func TestDatabaseStore_batchUpdateDeletedFlag_EmptyCodes(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	userID := "test-user"
	var codes []model.Code

	// Act
	err := store.batchUpdateDeletedFlag(context.Background(), codes, userID, true)

	// Assert
	assert.NoError(t, err)
}
