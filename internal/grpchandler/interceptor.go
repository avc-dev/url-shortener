// Package grpchandler реализует gRPC-хендлеры сервиса сокращения URL.
package grpchandler

import (
	"context"
	"strings"

	"github.com/avc-dev/url-shortener/internal/middleware"
	"github.com/avc-dev/url-shortener/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// authenticatedKey — контекстный ключ, сигнализирующий о том, что клиент передал
// валидный JWT-токен. Отдельный от UserIDContextKey, чтобы различать
// «анонимный пользователь» и «аутентифицированный пользователь».
type authenticatedKey struct{}

// AuthInterceptor возвращает unary-интерцептор, который:
//   - Извлекает JWT из metadata-заголовка "authorization" (поддерживает формат "Bearer <token>" и просто токен).
//   - При валидном токене устанавливает user_id в контекст и помечает запрос как аутентифицированный.
//   - При отсутствующем или невалидном токене генерирует анонимный user_id (для методов, не требующих авторизации).
func AuthInterceptor(authService *service.AuthService) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		userID, authenticated := extractUserID(ctx, authService)
		ctx = context.WithValue(ctx, middleware.UserIDContextKey, userID)
		ctx = context.WithValue(ctx, authenticatedKey{}, authenticated)
		return handler(ctx, req)
	}
}

// IsAuthenticated возвращает true, если запрос был выполнен с валидным JWT-токеном.
func IsAuthenticated(ctx context.Context) bool {
	v, _ := ctx.Value(authenticatedKey{}).(bool)
	return v
}

// extractUserID пытается извлечь user_id из JWT в metadata.
// Возвращает (userID, true) при успехе и (новый_userID, false) при неудаче.
func extractUserID(ctx context.Context, authService *service.AuthService) (string, bool) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get("authorization"); len(values) > 0 {
			token := strings.TrimPrefix(values[0], "Bearer ")
			if userID, err := authService.ValidateJWT(token); err == nil {
				return userID, true
			}
		}
	}
	return authService.GenerateUserID(), false
}
