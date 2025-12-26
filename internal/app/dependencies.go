package app

import (
	"context"
	"fmt"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/config/db"
	"github.com/avc-dev/url-shortener/internal/handler"
	"github.com/avc-dev/url-shortener/internal/migrations"
	"github.com/avc-dev/url-shortener/internal/repository"
	"github.com/avc-dev/url-shortener/internal/service"
	"github.com/avc-dev/url-shortener/internal/store"
	"github.com/avc-dev/url-shortener/internal/usecase"
	"go.uber.org/zap"
)

// initDependencies инициализирует все зависимости приложения
func initDependencies(cfg *config.Config, logger *zap.Logger) (*handler.Handler, db.Database, error) {
	var dbPool db.Database
	if cfg.DatabaseDSN != "" {
		var err error
		dbPool, err = initDatabase(cfg, logger)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize database: %w", err)
		}
	}

	storage, err := initStorage(cfg, dbPool, logger)
	if err != nil {
		if dbPool != nil {
			dbPool.Close()
		}
		return nil, nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	repo := repository.New(storage)
	urlService := service.NewURLService(repo, cfg)
	authService := service.NewAuthService(cfg.JWTSecret)
	urlUsecase := usecase.NewURLUsecase(repo, urlService, cfg, logger)
	h := handler.New(urlUsecase, logger, dbPool, authService)

	return h, dbPool, nil
}

// initDatabase инициализирует подключение к базе данных и применяет миграции
func initDatabase(cfg *config.Config, logger *zap.Logger) (db.Database, error) {
	ctx := context.Background()
	dbConfig := db.NewConfig(cfg.DatabaseDSN)

	pool, err := dbConfig.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	logger.Info("Connected to database", zap.String("dsn", cfg.DatabaseDSN))

	// Применяем миграции автоматически
	migrator := migrations.NewMigrator(pool.DB(), logger)
	if err := migrator.RunUp(); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	return pool, nil
}

// initStorage создает хранилище на основе конфигурации с приоритетом:
// 1. PostgreSQL (если доступна БД)
// 2. File storage (если указан путь к файлу)
// 3. In-memory storage
func initStorage(cfg *config.Config, dbPool db.Database, logger *zap.Logger) (repository.Store, error) {
	// Приоритет 1: PostgreSQL
	if dbPool != nil {
		logger.Info("Using PostgreSQL storage")
		return store.NewDatabaseStore(dbPool), nil
	}

	// Приоритет 2: File storage
	if cfg.FileStoragePath != "" {
		fileStore, err := store.NewFileStore(cfg.FileStoragePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create file store: %w", err)
		}
		logger.Info("Using file storage", zap.String("path", cfg.FileStoragePath))
		return fileStore, nil
	}

	// Приоритет 3: In-memory storage
	logger.Info("Using in-memory storage")
	return store.NewStore(), nil
}
