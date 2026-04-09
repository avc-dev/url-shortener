// Package app отвечает за сборку и запуск приложения.
// App объединяет конфигурацию, хендлер, базу данных, сервис аутентификации
// и систему аудита в единый компонент жизненного цикла.
package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/avc-dev/url-shortener/internal/audit"
	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/config/db"
	"github.com/avc-dev/url-shortener/internal/handler"
	"github.com/avc-dev/url-shortener/internal/service"
	"github.com/avc-dev/url-shortener/internal/usecase"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

// App представляет приложение URL shortener
type App struct {
	config      *config.Config
	logger      *zap.Logger
	handler     *handler.Handler
	dbPool      db.Database
	authService *service.AuthService
	urlUsecase  *usecase.URLUsecase
	audit       *audit.Subject
	httpServer  *http.Server
	grpcServer  *grpc.Server
	healthSrv   *health.Server
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

	h, dbPool, authService, auditSubject, urlUsecase, grpcSrv, healthSrv, err := initDependencies(cfg, logger)
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
		urlUsecase:  urlUsecase,
		audit:       auditSubject,
		grpcServer:  grpcSrv,
		healthSrv:   healthSrv,
	}, nil
}

// Run — точка входа для запуска сервера из main.
// Запускает HTTP и gRPC серверы параллельно, обрабатывает сигналы SIGTERM/SIGINT/SIGQUIT
// для graceful shutdown.
func Run() error {
	app, err := New()
	if err != nil {
		return err
	}
	defer app.logger.Sync()

	httpLn, grpcLn, err := app.prepare()
	if err != nil {
		app.Close()
		return err
	}

	// Health checker живёт пока серверы работают.
	// Отменяется первым — до shutdown, чтобы не обновлять статус в процессе остановки.
	healthCtx, cancelHealth := context.WithCancel(context.Background())
	defer cancelHealth()
	app.startHealthChecker(healthCtx)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer signal.Stop(sigCh)

	errCh := make(chan error, 2)
	go func() { errCh <- app.serveHTTP(httpLn) }()
	go func() { errCh <- app.serveGRPC(grpcLn) }()

	select {
	case sig := <-sigCh:
		app.logger.Info("Received signal, shutting down", zap.String("signal", sig.String()))
		cancelHealth()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if shutdownErr := app.shutdown(ctx); shutdownErr != nil {
			app.logger.Error("Graceful shutdown failed", zap.Error(shutdownErr))
		}
		app.Close()
		return nil

	case err := <-errCh:
		// Один из серверов упал — останавливаем второй перед выходом,
		// чтобы in-flight запросы не были обрублены внезапно.
		app.logger.Error("Server failed, initiating shutdown", zap.Error(err))
		cancelHealth()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if shutdownErr := app.shutdown(shutdownCtx); shutdownErr != nil {
			app.logger.Error("Graceful shutdown after failure failed", zap.Error(shutdownErr))
		}
		app.Close()
		return err
	}
}

// Close освобождает ресурсы приложения в безопасном порядке:
// 1. Ждёт завершения горутин удаления URL (работают с БД).
// 2. Ждёт завершения горутин аудита (работают с файлом/сетью).
// 3. Закрывает пул соединений с БД.
func (a *App) Close() {
	if a.urlUsecase != nil {
		a.urlUsecase.Close()
	}
	if a.audit != nil {
		a.audit.Close()
	}
	if a.dbPool != nil {
		a.dbPool.Close()
		a.logger.Info("Database connection pool closed")
	}
}
