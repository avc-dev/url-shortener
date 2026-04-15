package app

import (
	"context"
	"time"

	pb "github.com/avc-dev/url-shortener/internal/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	healthCheckInterval = 30 * time.Second
	healthCheckTimeout  = 5 * time.Second
)

// startHealthChecker запускает фоновую горутину, которая периодически пингует БД
// и обновляет статус gRPC health check сервера. Горутина завершается при отмене ctx.
//
// Если БД не настроена (in-memory / file storage), проверка не нужна:
// эти хранилища не имеют внешних зависимостей, которые могут упасть.
func (a *App) startHealthChecker(ctx context.Context) {
	if a.healthSrv == nil || a.dbPool == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(healthCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.syncHealthStatus()
			}
		}
	}()
}

// syncHealthStatus пингует БД и обновляет статусы health check сервера.
// Использует собственный контекст с таймаутом, независимый от контекста приложения:
// пинг должен завершиться корректно даже если приложение начало остановку.
func (a *App) syncHealthStatus() {
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	svcName := pb.ShortenerService_ServiceDesc.ServiceName
	if err := a.dbPool.Ping(ctx); err != nil {
		a.healthSrv.SetServingStatus(svcName, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		a.healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		a.logger.Warn("health check: database unreachable, marking NOT_SERVING", zap.Error(err))
		return
	}
	a.healthSrv.SetServingStatus(svcName, grpc_health_v1.HealthCheckResponse_SERVING)
	a.healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
}
