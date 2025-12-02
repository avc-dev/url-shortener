package app

import (
	"net/http"

	"go.uber.org/zap"
)

// start запускает HTTP сервер
func (a *App) start() error {
	router := newRouter(a.handler, a.logger)

	a.logger.Info("Starting server", zap.String("address", a.config.ServerAddress.String()))

	err := http.ListenAndServe(a.config.ServerAddress.String(), router)
	if err != nil {
		a.logger.Fatal("Server failed", zap.Error(err))
		return err
	}

	return nil
}

