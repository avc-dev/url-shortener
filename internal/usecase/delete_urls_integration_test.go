package usecase

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/config/db"
	"github.com/avc-dev/url-shortener/internal/migrations"
	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/avc-dev/url-shortener/internal/repository"
	"github.com/avc-dev/url-shortener/internal/service"
	"github.com/avc-dev/url-shortener/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupIntegrationTestDB создает тестовую базу данных для интеграционных тестов
func setupIntegrationTestDB(t *testing.T) (*URLUsecase, func()) {
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

	// Создаем зависимости
	cfg := &config.Config{}
	databaseStore := store.NewDatabaseStore(database)
	repo := repository.New(databaseStore)
	urlService := service.NewURLService(repo, cfg)

	// Создаем usecase
	usecase := NewURLUsecase(repo, urlService, cfg, logger)

	// Очищаем таблицы перед каждым тестом
	_, err = database.DB().Exec("DELETE FROM urls")
	require.NoError(t, err)

	// Возвращаем usecase и cleanup функцию
	cleanup := func() {
		database.Close()
	}

	return usecase, cleanup
}

// TestDeleteURLsIntegration_FullCycle проверяет полный цикл создания, удаления и проверки состояния URL
func TestDeleteURLsIntegration_FullCycle(t *testing.T) {
	usecase, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	userID := "test-user"
	originalURLs := []string{
		"https://example1.com",
		"https://example2.com",
		"https://example3.com",
	}

	// Создаем короткие URL
	codes := make([]string, len(originalURLs))
	for i, url := range originalURLs {
		code, _, err := usecase.service.CreateShortURL(model.URL(url), userID)
		require.NoError(t, err)
		codes[i] = string(code)

		// Сохраняем URL в базе данных
		_, _, err = usecase.repo.CreateOrGetURL(code, model.URL(url), userID)
		require.NoError(t, err)
	}

	// Проверяем что URL созданы и доступны
	for _, code := range codes {
		url, err := usecase.repo.GetURLByCode(model.Code(code))
		require.NoError(t, err)
		assert.NotEmpty(t, url)
	}

	// Act - удаляем URL
	err := usecase.DeleteURLs(codes, userID)
	require.NoError(t, err)

	// Даем время асинхронной операции завершиться
	time.Sleep(200 * time.Millisecond)

	// Assert - проверяем что URL помечены как удаленные
	for _, code := range codes {
		_, err := usecase.repo.GetURLByCode(model.Code(code))
		assert.Error(t, err, "URL should be marked as deleted")
	}
}

// TestDeleteURLsIntegration_PartialDeletion проверяет удаление только принадлежащих пользователю URL
func TestDeleteURLsIntegration_PartialDeletion(t *testing.T) {
	usecase, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	userID1 := "user1"
	userID2 := "user2"

	// Создаем URL для разных пользователей
	code1, _, err := usecase.service.CreateShortURL(model.URL("https://user1.com"), userID1)
	require.NoError(t, err)
	_, _, err = usecase.repo.CreateOrGetURL(code1, model.URL("https://user1.com"), userID1)
	require.NoError(t, err)

	code2, _, err := usecase.service.CreateShortURL(model.URL("https://user2.com"), userID2)
	require.NoError(t, err)
	_, _, err = usecase.repo.CreateOrGetURL(code2, model.URL("https://user2.com"), userID2)
	require.NoError(t, err)

	code3, _, err := usecase.service.CreateShortURL(model.URL("https://user1-2.com"), userID1)
	require.NoError(t, err)
	_, _, err = usecase.repo.CreateOrGetURL(code3, model.URL("https://user1-2.com"), userID1)
	require.NoError(t, err)

	// user1 пытается удалить все URL (включая принадлежащий user2)
	allCodes := []string{string(code1), string(code2), string(code3)}
	err = usecase.DeleteURLs(allCodes, userID1)
	require.NoError(t, err)

	// Даем время асинхронной операции завершиться
	time.Sleep(200 * time.Millisecond)

	// Проверяем состояние URL
	_, err = usecase.repo.GetURLByCode(code1) // URL user1 - должен быть удален
	assert.Error(t, err, "user1's URL should be deleted")

	_, err = usecase.repo.GetURLByCode(code2) // URL user2 - не должен быть удален
	assert.NoError(t, err, "user2's URL should not be deleted")

	_, err = usecase.repo.GetURLByCode(code3) // URL user1 - должен быть удален
	assert.Error(t, err, "user1's second URL should be deleted")
}

// TestDeleteURLsIntegration_ConcurrentDeletion проверяет конкурентное удаление URL
func TestDeleteURLsIntegration_ConcurrentDeletion(t *testing.T) {
	usecase, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	userID := "test-user"
	numURLs := 10

	// Создаем несколько URL
	codes := make([]string, numURLs)
	for i := 0; i < numURLs; i++ {
		url := fmt.Sprintf("https://example%d.com", i)
		code, _, err := usecase.service.CreateShortURL(model.URL(url), userID)
		require.NoError(t, err)
		codes[i] = string(code)

		// Сохраняем URL в базе данных
		_, _, err = usecase.repo.CreateOrGetURL(code, model.URL(url), userID)
		require.NoError(t, err)
	}

	// Разделяем коды на две партии для конкурентного удаления
	half := numURLs / 2
	codes1 := codes[:half]
	codes2 := codes[half:]

	// Act - запускаем два конкурентных удаления
	done := make(chan bool, 2)

	go func() {
		err := usecase.DeleteURLs(codes1, userID)
		assert.NoError(t, err)
		done <- true
	}()

	go func() {
		err := usecase.DeleteURLs(codes2, userID)
		assert.NoError(t, err)
		done <- true
	}()

	// Ждем завершения обоих операций
	<-done
	<-done

	// Даем время асинхронным операциям завершиться
	time.Sleep(500 * time.Millisecond)

	// Assert - проверяем что все URL удалены
	for _, code := range codes {
		_, err := usecase.repo.GetURLByCode(model.Code(code))
		assert.Error(t, err, "URL %s should be deleted", code)
	}
}

// TestDeleteURLsIntegration_EmptyCodes проверяет обработку пустого списка кодов
func TestDeleteURLsIntegration_EmptyCodes(t *testing.T) {
	usecase, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	userID := "test-user"
	var codes []string

	// Act
	err := usecase.DeleteURLs(codes, userID)

	// Assert
	assert.NoError(t, err)
}

// TestDeleteURLsIntegration_NonExistentCodes проверяет обработку несуществующих кодов
func TestDeleteURLsIntegration_NonExistentCodes(t *testing.T) {
	usecase, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	userID := "test-user"
	codes := []string{"nonexistent1", "nonexistent2", "nonexistent3"}

	// Act
	err := usecase.DeleteURLs(codes, userID)

	// Assert
	assert.NoError(t, err)

	// Даем время асинхронной операции завершиться
	time.Sleep(200 * time.Millisecond)

	// Проверяем что несуществующие коды не вызвали ошибок
	// (они просто не прошли валидацию принадлежности пользователю)
}

// TestDeleteURLsIntegration_LargeBatch проверяет удаление большого количества URL
func TestDeleteURLsIntegration_LargeBatch(t *testing.T) {
	usecase, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	userID := "test-user"
	numURLs := 50

	// Создаем много URL
	codes := make([]string, numURLs)
	for i := 0; i < numURLs; i++ {
		url := fmt.Sprintf("https://large-batch%d.com", i)
		code, _, err := usecase.service.CreateShortURL(model.URL(url), userID)
		require.NoError(t, err)
		codes[i] = string(code)

		// Сохраняем URL в базе данных
		_, _, err = usecase.repo.CreateOrGetURL(code, model.URL(url), userID)
		require.NoError(t, err)
	}

	// Act - удаляем все URL
	err := usecase.DeleteURLs(codes, userID)
	require.NoError(t, err)

	// Даем время асинхронной операции завершиться
	time.Sleep(500 * time.Millisecond)

	// Assert - проверяем что все URL удалены
	for _, code := range codes {
		_, err := usecase.repo.GetURLByCode(model.Code(code))
		assert.Error(t, err, "URL %s should be deleted", code)
	}
}
