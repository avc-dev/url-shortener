package app

import (
	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/handler"
	"go.uber.org/zap"
)

// App представляет приложение URL shortener
type App struct {
	config  *config.Config
	logger  *zap.Logger
	handler *handler.Handler
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

	h, err := initDependencies(cfg, logger)
	if err != nil {
		logger.Sync()
		return nil, err
	}

	return &App{
		config:  cfg,
		logger:  logger,
		handler: h,
	}, nil
}

// Run запускает приложение
func Run() error {
	app, err := New()
	if err != nil {
		return err
	}
	defer app.logger.Sync()

	return app.start()
}
