package service

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthService предоставляет функциональность для аутентификации пользователей
type AuthService struct {
	jwtSecret []byte
}

// NewAuthService создает новый экземпляр AuthService
func NewAuthService(jwtSecret string) *AuthService {
	return &AuthService{
		jwtSecret: []byte(jwtSecret),
	}
}

// GenerateUserID генерирует уникальный идентификатор пользователя
func (a *AuthService) GenerateUserID() string {
	return uuid.New().String()
}

// GenerateJWT создает JWT токен для пользователя
func (a *AuthService) GenerateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(), // токен действителен 24 часа
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

// ValidateJWT проверяет JWT токен и извлекает user_id
func (a *AuthService) ValidateJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if userID, ok := claims["user_id"].(string); ok {
			return userID, nil
		}
		return "", fmt.Errorf("user_id not found in token")
	}

	return "", fmt.Errorf("invalid token")
}

// GetOrCreateUserFromCookie извлекает user_id из куки или создает нового пользователя
func (a *AuthService) GetOrCreateUserFromCookie(r *http.Request, w http.ResponseWriter) (string, error) {
	cookie, err := r.Cookie("user_token")
	if err != nil || cookie.Value == "" {
		// Куки нет или пустая, создаем нового пользователя
		userID := a.GenerateUserID()
		token, err := a.GenerateJWT(userID)
		if err != nil {
			return "", fmt.Errorf("failed to generate JWT: %w", err)
		}

		// Устанавливаем куку
		http.SetCookie(w, &http.Cookie{
			Name:     "user_token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // для разработки, в продакшене должен быть true
			SameSite: http.SameSiteLaxMode,
			MaxAge:   86400, // 24 часа
		})

		return userID, nil
	}

	// Куки есть, проверяем токен
	userID, err := a.ValidateJWT(cookie.Value)
	if err != nil {
		// Токен недействителен, создаем нового пользователя
		userID := a.GenerateUserID()
		token, err := a.GenerateJWT(userID)
		if err != nil {
			return "", fmt.Errorf("failed to generate JWT: %w", err)
		}

		// Обновляем куку
		http.SetCookie(w, &http.Cookie{
			Name:     "user_token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   86400,
		})

		return userID, nil
	}

	return userID, nil
}
