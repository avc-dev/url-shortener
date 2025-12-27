package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestDeleteURLs_Success проверяет успешное удаление URL
func TestDeleteURLs_Success(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	mockService := mocks.NewMockURLService(t)
	cfg := &config.Config{}
	logger := zap.NewNop()
	done := make(chan struct{}, 1)

	usecase := NewURLUsecaseWithDone(mockRepo, mockService, cfg, logger, done)

	codes := []string{"abc123", "def456"}
	userID := "test-user"

	// Мокаем валидацию принадлежности URL пользователю
	mockRepo.EXPECT().
		IsURLOwnedByUser(model.Code("abc123"), userID).
		Return(true).
		Once()
	mockRepo.EXPECT().
		IsURLOwnedByUser(model.Code("def456"), userID).
		Return(true).
		Once()

	// Мокаем batch удаление (порядок может быть любым из-за параллельной обработки)
	mockRepo.EXPECT().
		DeleteURLsBatch(mock.MatchedBy(func(codes []model.Code) bool {
			if len(codes) != 2 {
				return false
			}
			codeMap := make(map[model.Code]bool)
			for _, code := range codes {
				codeMap[code] = true
			}
			return codeMap[model.Code("abc123")] && codeMap[model.Code("def456")]
		}), userID).
		Return(nil).
		Once()

	// Act
	err := usecase.DeleteURLs(codes, userID)

	// Assert
	require.NoError(t, err)

	// Ждем завершения асинхронной операции с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	select {
	case <-done:
		// Операция завершилась успешно
	case <-ctx.Done():
		t.Fatal("Тест завис: асинхронная операция не завершилась за 1 секунду")
	}
	close(done)
}

// TestDeleteURLs_NoValidCodes проверяет случай когда ни один URL не принадлежит пользователю
func TestDeleteURLs_NoValidCodes(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	mockService := mocks.NewMockURLService(t)
	cfg := &config.Config{}
	logger := zap.NewNop()
	done := make(chan struct{}, 1)

	usecase := NewURLUsecaseWithDone(mockRepo, mockService, cfg, logger, done)

	codes := []string{"abc123", "def456"}
	userID := "test-user"

	// Мокаем валидацию - ни один URL не принадлежит пользователю
	mockRepo.EXPECT().
		IsURLOwnedByUser(model.Code("abc123"), userID).
		Return(false).
		Once()
	mockRepo.EXPECT().
		IsURLOwnedByUser(model.Code("def456"), userID).
		Return(false).
		Once()

	// Act
	err := usecase.DeleteURLs(codes, userID)

	// Assert
	require.NoError(t, err)

	// DeleteURLsBatch не должен вызываться, так как нет валидных кодов
	mockRepo.AssertNotCalled(t, "DeleteURLsBatch")

	// Ждем завершения асинхронной операции
	<-done
	close(done)
}

// TestDeleteURLs_PartialValidCodes проверяет случай когда только некоторые URL принадлежат пользователю
func TestDeleteURLs_PartialValidCodes(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	mockService := mocks.NewMockURLService(t)
	cfg := &config.Config{}
	logger := zap.NewNop()
	done := make(chan struct{}, 1)

	usecase := NewURLUsecaseWithDone(mockRepo, mockService, cfg, logger, done)

	codes := []string{"abc123", "def456", "xyz789"}
	userID := "test-user"

	// Мокаем валидацию - только первый и третий URL принадлежат пользователю
	mockRepo.EXPECT().
		IsURLOwnedByUser(model.Code("abc123"), userID).
		Return(true).
		Once()
	mockRepo.EXPECT().
		IsURLOwnedByUser(model.Code("def456"), userID).
		Return(false).
		Once()
	mockRepo.EXPECT().
		IsURLOwnedByUser(model.Code("xyz789"), userID).
		Return(true).
		Once()

	// Мокаем batch удаление только для валидных кодов (порядок может быть любым из-за асинхронной обработки)
	mockRepo.EXPECT().
		DeleteURLsBatch(mock.AnythingOfType("[]model.Code"), userID).
		Return(nil).
		Once()

	// Act
	err := usecase.DeleteURLs(codes, userID)

	// Assert
	require.NoError(t, err)

	// Ждем завершения асинхронной операции с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	select {
	case <-done:
		// Операция завершилась успешно
	case <-ctx.Done():
		t.Fatal("Тест завис: асинхронная операция не завершилась за 1 секунду")
	}
	close(done)
}

// TestDeleteURLs_EmptyCodes проверяет обработку пустого списка кодов
func TestDeleteURLs_EmptyCodes(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	mockService := mocks.NewMockURLService(t)
	cfg := &config.Config{}
	logger := zap.NewNop()
	done := make(chan struct{}, 1)

	usecase := NewURLUsecaseWithDone(mockRepo, mockService, cfg, logger, done)

	codes := []string{}
	userID := "test-user"

	// Act
	err := usecase.DeleteURLs(codes, userID)

	// Assert
	require.NoError(t, err)

	// Никакие методы репозитория не должны вызываться
	mockRepo.AssertNotCalled(t, "IsURLOwnedByUser")
	mockRepo.AssertNotCalled(t, "DeleteURLsBatch")

	// Ждем завершения асинхронной операции
	<-done
	close(done)
}

// TestDeleteURLs_DeleteBatchError проверяет обработку ошибки при batch удалении
func TestDeleteURLs_DeleteBatchError(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	mockService := mocks.NewMockURLService(t)
	cfg := &config.Config{}
	logger := zap.NewNop()
	done := make(chan struct{}, 1)

	usecase := NewURLUsecaseWithDone(mockRepo, mockService, cfg, logger, done)

	codes := []string{"abc123"}
	userID := "test-user"

	// Мокаем валидацию
	mockRepo.EXPECT().
		IsURLOwnedByUser(model.Code("abc123"), userID).
		Return(true).
		Once()

	// Мокаем ошибку при batch удалении
	mockRepo.EXPECT().
		DeleteURLsBatch(mock.MatchedBy(func(codes []model.Code) bool {
			return len(codes) == 1 && codes[0] == model.Code("abc123")
		}), userID).
		Return(assert.AnError).
		Once()

	// Act
	err := usecase.DeleteURLs(codes, userID)

	// Assert
	require.NoError(t, err) // DeleteURLs всегда возвращает nil, ошибки логируются

	// Ждем завершения асинхронной операции
	<-done
	close(done)
}

// TestDeleteURLs_SingleCode проверяет удаление одного URL
func TestDeleteURLs_SingleCode(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	mockService := mocks.NewMockURLService(t)
	cfg := &config.Config{}
	logger := zap.NewNop()
	done := make(chan struct{}, 1)

	usecase := NewURLUsecaseWithDone(mockRepo, mockService, cfg, logger, done)

	codes := []string{"single123"}
	userID := "test-user"

	// Мокаем валидацию
	mockRepo.EXPECT().
		IsURLOwnedByUser(model.Code("single123"), userID).
		Return(true).
		Once()

	// Мокаем batch удаление
	mockRepo.EXPECT().
		DeleteURLsBatch(mock.MatchedBy(func(codes []model.Code) bool {
			return len(codes) == 1 && codes[0] == model.Code("single123")
		}), userID).
		Return(nil).
		Once()

	// Act
	err := usecase.DeleteURLs(codes, userID)

	// Assert
	require.NoError(t, err)

	// Ждем завершения асинхронной операции с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	select {
	case <-done:
		// Операция завершилась успешно
	case <-ctx.Done():
		t.Fatal("Тест завис: асинхронная операция не завершилась за 1 секунду")
	}
	close(done)
}

// TestDeleteURLs_LargeBatch проверяет удаление большого количества URL
func TestDeleteURLs_LargeBatch(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockURLRepository(t)
	mockService := mocks.NewMockURLService(t)
	cfg := &config.Config{}
	logger := zap.NewNop()
	done := make(chan struct{}, 1)

	usecase := NewURLUsecaseWithDone(mockRepo, mockService, cfg, logger, done)

	// Создаем большой список кодов
	codes := make([]string, 100)
	userID := "test-user"

	for i := 0; i < 100; i++ {
		code := model.Code(string(rune('a'+i%26)) + string(rune('0'+i%10)))
		codes[i] = string(code)

		// Мокаем валидацию для каждого кода
		mockRepo.EXPECT().
			IsURLOwnedByUser(code, userID).
			Return(true).
			Once()
	}

	// Мокаем batch удаление - принимаем массив из 100 элементов
	mockRepo.EXPECT().
		DeleteURLsBatch(mock.MatchedBy(func(codes []model.Code) bool {
			return len(codes) == 100 // Просто проверяем размер массива
		}), userID).
		Return(nil).
		Once()

	// Act
	err := usecase.DeleteURLs(codes, userID)

	// Assert
	require.NoError(t, err)

	// Ждем завершения асинхронной операции с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	select {
	case <-done:
		// Операция завершилась успешно
	case <-ctx.Done():
		t.Fatal("Тест завис: асинхронная операция не завершилась за 1 секунду")
	}
	close(done)
}
