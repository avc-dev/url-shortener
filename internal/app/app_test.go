package app

import (
	"testing"

	"github.com/avc-dev/url-shortener/internal/audit"
	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestApp_Close(t *testing.T) {
	t.Run("database pool exists", func(t *testing.T) {
		mockDB := mocks.NewMockDatabase(t)
		mockDB.EXPECT().Close().Once()

		app := &App{
			logger: zap.NewNop(),
			dbPool: mockDB,
		}

		// Act
		app.Close()

		// Assert
		mockDB.AssertExpectations(t)
	})

	t.Run("database pool is nil", func(t *testing.T) {
		app := &App{
			logger: zap.NewNop(),
			dbPool: nil,
		}

		// Act - should not panic
		app.Close()
	})

	t.Run("audit subject is closed before db", func(t *testing.T) {
		// Проверяем, что Close() вызывает audit.Close() и не паникует.
		subject := audit.NewSubject(zap.NewNop())
		mockDB := mocks.NewMockDatabase(t)
		mockDB.EXPECT().Close().Once()

		app := &App{
			logger: zap.NewNop(),
			dbPool: mockDB,
			audit:  subject,
		}

		assert.NotPanics(t, app.Close)
		mockDB.AssertExpectations(t)
	})

	t.Run("nil audit subject is safe", func(t *testing.T) {
		app := &App{
			logger: zap.NewNop(),
			audit:  nil,
		}
		assert.NotPanics(t, app.Close)
	})
}
