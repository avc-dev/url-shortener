package middleware

import (
	"context"
	"net/http"

	"github.com/avc-dev/url-shortener/internal/service"
	"go.uber.org/zap"
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
		ctx := context.WithValue(r.Context(), "user_id", userID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// RequireAuth возвращает миддлвар, который требует валидной аутентификации
// Если пользователь не аутентифицирован, возвращает 401 Unauthorized
func (am *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("user_token")
		if err != nil || cookie.Value == "" {
			am.logger.Debug("no auth cookie found")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userID, err := am.authService.ValidateJWT(cookie.Value)
		if err != nil {
			am.logger.Debug("invalid auth token", zap.Error(err))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Добавляем user_id в контекст
		ctx := context.WithValue(r.Context(), "user_id", userID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// GetUserIDFromContext извлекает user_id из контекста запроса
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value("user_id").(string)
	return userID, ok
}

