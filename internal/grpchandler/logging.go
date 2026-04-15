package grpchandler

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// clientCodes — gRPC-коды, которые являются ожидаемыми бизнес-ситуациями
// (ошибка на стороне клиента). Не требуют алерта, логируются как Warn.
var clientCodes = map[codes.Code]bool{
	codes.InvalidArgument:   true,
	codes.NotFound:          true,
	codes.AlreadyExists:     true,
	codes.Unauthenticated:   true,
	codes.PermissionDenied:  true,
	codes.ResourceExhausted: true,
}

// LoggingInterceptor возвращает unary-интерцептор, который логирует каждый gRPC-запрос.
// Уровень логирования зависит от типа ошибки:
//   - OK                         → Debug  (успешный запрос, не засоряет Info-лог)
//   - клиентские коды (4xx-аналог) → Info   (ожидаемые бизнес-ситуации)
//   - серверные коды  (5xx-аналог) → Error  (аномалии, требуют расследования)
func LoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		code := status.Code(err)
		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", time.Since(start)),
			zap.String("code", code.String()),
		}

		switch {
		case code == codes.OK:
			logger.Debug("gRPC request", fields...)
		case clientCodes[code]:
			logger.Info("gRPC request", fields...)
		default:
			logger.Error("gRPC request", append(fields, zap.Error(err))...)
		}

		return resp, err
	}
}
