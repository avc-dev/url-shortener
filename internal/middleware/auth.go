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

// RequireAuth возвращает миддлвар для анонимной аутентификации
// Для совместимости с тестами - если куки нет, использует пустой userID
func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var userID string

		cookie, err := r.Cookie("user_token")
		if err != nil || cookie.Value == "" {
			// Куки нет - используем пустой userID (анонимный пользователь)
			am.logger.Debug("no auth cookie found, proceeding with empty userID")
			userID = ""
		} else {
			// Куки есть - проверяем токен
			userID, err = am.authService.ValidateJWT(cookie.Value)
			if err != nil {
				am.logger.Debug("invalid auth token", zap.Error(err))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		// Добавляем user_id в контекст
		ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// OptionalAuth возвращает миддлвар для опциональной аутентификации
// Если куки нет - работает анонимно (не устанавливает userID)
// Если кука есть и валидная - устанавливает userID в контекст
// Если кука есть но невалидная - возвращает 401 Unauthorized
func (am *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("user_token")
		if err != nil || cookie.Value == "" {
			// Куки нет - работаем анонимно, userID останется пустым
			am.logger.Debug("no auth cookie found, proceeding anonymously")
			next.ServeHTTP(w, r)
			return
		}

		userID, err := am.authService.ValidateJWT(cookie.Value)
		if err != nil {
			am.logger.Debug("invalid auth token", zap.Error(err))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Куки валидная - добавляем user_id в контекст
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
