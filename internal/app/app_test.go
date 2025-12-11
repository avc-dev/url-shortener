package app

import (
	"testing"

	"github.com/avc-dev/url-shortener/internal/mocks"
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

		// Assert - no assertions needed, just ensure no panic
	})
}
