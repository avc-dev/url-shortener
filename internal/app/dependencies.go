package app

import (
	"context"
	"fmt"

	"github.com/avc-dev/url-shortener/internal/audit"
	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/config/db"
	"github.com/avc-dev/url-shortener/internal/grpchandler"
	"github.com/avc-dev/url-shortener/internal/handler"
	"github.com/avc-dev/url-shortener/internal/migrations"
	pb "github.com/avc-dev/url-shortener/internal/proto"
	"github.com/avc-dev/url-shortener/internal/repository"
	"github.com/avc-dev/url-shortener/internal/service"
	"github.com/avc-dev/url-shortener/internal/store"
	"github.com/avc-dev/url-shortener/internal/usecase"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// initDependencies инициализирует все зависимости приложения
func initDependencies(cfg *config.Config, logger *zap.Logger) (*handler.Handler, db.Database, *service.AuthService, *audit.Subject, *usecase.URLUsecase, *grpc.Server, error) {
	var dbPool db.Database
	if cfg.DatabaseDSN != "" {
		var err error
		dbPool, err = initDatabase(cfg, logger)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to initialize database: %w", err)
		}
	}

	storage, err := initStorage(cfg, dbPool, logger)
	if err != nil {
		if dbPool != nil {
			dbPool.Close()
		}
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	repo := repository.New(storage)
	urlService := service.NewURLService(repo, cfg)
	authService := service.NewAuthService(cfg.JWTSecret)
	urlUsecase := usecase.NewURLUsecase(repo, urlService, cfg, logger)

	auditSubject := initAudit(cfg, logger)

	// Передаём Subject в Handler только если он не nil, чтобы избежать
	// typed-nil внутри интерфейса handler.Auditor.
	var handlerOpts []handler.Auditor
	if auditSubject != nil {
		handlerOpts = append(handlerOpts, auditSubject)
	}
	h := handler.New(urlUsecase, logger, dbPool, handlerOpts...)

	grpcSrv := initGRPCServer(urlUsecase, authService, auditSubject, logger)

	return h, dbPool, authService, auditSubject, urlUsecase, grpcSrv, nil
}

// initGRPCServer создаёт gRPC-сервер с chain-интерцепторами и регистрирует:
//   - ShortenerService — основной бизнес-хендлер
//   - Health — стандартный health check (grpc_health_v1)
//   - Reflection — для grpcurl и Postman без .proto файла
func initGRPCServer(urlUsecase *usecase.URLUsecase, authService *service.AuthService, auditSubject *audit.Subject, logger *zap.Logger) *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpchandler.LoggingInterceptor(logger),
			grpchandler.AuthInterceptor(authService),
		),
	)

	// Основной бизнес-хендлер — с аудитом, если он настроен.
	var auditors []grpchandler.Auditor
	if auditSubject != nil {
		auditors = append(auditors, auditSubject)
	}
	pb.RegisterShortenerServiceServer(srv, grpchandler.New(urlUsecase, auditors...))

	// Health check — сигнализируем SERVING для всего сервера и конкретного сервиса.
	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus(pb.ShortenerService_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(srv, healthSrv)

	// Reflection — позволяет grpcurl/Postman работать без .proto файла.
	reflection.Register(srv)

	return srv
}

// initAudit создаёт Subject с наблюдателями на основе конфигурации.
// Возвращает nil, если ни один приёмник аудита не настроен.
func initAudit(cfg *config.Config, logger *zap.Logger) *audit.Subject {
	if cfg.AuditFile == "" && cfg.AuditURL == "" {
		return nil
	}

	subject := audit.NewSubject(logger)
	if cfg.AuditFile != "" {
		subject.Register(audit.NewFileObserver(cfg.AuditFile))
		logger.Info("Audit file sink enabled", zap.String("path", cfg.AuditFile))
	}
	if cfg.AuditURL != "" {
		subject.Register(audit.NewRemoteObserver(cfg.AuditURL))
		logger.Info("Audit remote sink enabled", zap.String("url", cfg.AuditURL))
	}
	return subject
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
	if dbPool != nil {
		logger.Info("Using PostgreSQL storage")
		return store.NewDatabaseStore(dbPool), nil
	}

	if cfg.FileStoragePath != "" {
		fileStore, err := store.NewFileStore(cfg.FileStoragePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create file store: %w", err)
		}
		logger.Info("Using file storage", zap.String("path", cfg.FileStoragePath))
		return fileStore, nil
	}

	logger.Info("Using in-memory storage")
	return store.NewStore(), nil
}
