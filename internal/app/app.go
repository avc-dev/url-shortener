package app

import (
	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/config/db"
	"github.com/avc-dev/url-shortener/internal/handler"
	"github.com/avc-dev/url-shortener/internal/service"
	"go.uber.org/zap"
)

// App представляет приложение URL shortener
type App struct {
	config      *config.Config
	logger      *zap.Logger
	handler     *handler.Handler
	dbPool      db.Database
	authService *service.AuthService
}

// New создает новый экземпляр приложения
func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	h, dbPool, authService, err := initDependencies(cfg, logger)
	if err != nil {
		logger.Sync()
		return nil, err
	}

	return &App{
		config:      cfg,
		logger:      logger,
		handler:     h,
		dbPool:      dbPool,
		authService: authService,
	}, nil
}

// Run запускает приложение
func Run() error {
	app, err := New()
	if err != nil {
		return err
	}
	defer app.logger.Sync()
	defer app.Close()

	return app.start()
}

// Close закрывает ресурсы приложения
func (a *App) Close() {
	if a.dbPool != nil {
		a.dbPool.Close()
		a.logger.Info("Database connection pool closed")
	}
}
