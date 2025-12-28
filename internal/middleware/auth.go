package middleware

import (
	"context"
	"net/http"

	"github.com/avc-dev/url-shortener/internal/service"
	"go.uber.org/zap"
)

// UserIDKey is the key used to store user ID in context
type UserIDKey string

const (
	// UserIDContextKey is the context key for user ID
	UserIDContextKey UserIDKey = "user_id"
)

// AuthMiddleware представляет миддлвар для аутентификации пользователей
type AuthMiddleware struct {
	authService *service.AuthService
	logger      *zap.Logger
}

// NewAuthMiddleware создает новый экземпляр AuthMiddleware
func NewAuthMiddleware(authService *service.AuthService, logger *zap.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		logger:      logger,
	}
}

// Authenticate возвращает миддлвар, который аутентифицирует пользователя
// и добавляет user_id в контекст запроса
func (am *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := am.authService.GetOrCreateUserFromCookie(r, w)
		if err != nil {
			am.logger.Error("failed to authenticate user", zap.Error(err))
			http.Error(w, "Authentication failed", http.StatusInternalServerError)
			return
		}

		// Добавляем user_id в контекст
		ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// RequireAuth возвращает миддлвар, который требует аутентификации
// Всегда устанавливает уникальный userID (создает если нужно)
func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := am.authService.GetOrCreateUserFromCookie(r, w)
		if err != nil {
			am.logger.Error("failed to authenticate user", zap.Error(err))
			http.Error(w, "Authentication failed", http.StatusInternalServerError)
			return
		}

		// Добавляем user_id в контекст
		ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// OptionalAuth возвращает миддлвар для опциональной аутентификации
// Всегда устанавливает уникальный userID (создает если нужно)
func (am *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := am.authService.GetOrCreateUserFromCookie(r, w)
		if err != nil {
			am.logger.Error("failed to authenticate user", zap.Error(err))
			http.Error(w, "Authentication failed", http.StatusInternalServerError)
			return
		}

		// Добавляем user_id в контекст
		ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// GetUserIDFromContext извлекает user_id из контекста запроса
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(string)
	return userID, ok
}
