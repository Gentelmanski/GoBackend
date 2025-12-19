package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"student-backend/auth"
)

type AuthMiddleware struct {
	jwtService *auth.JWTService
}

func NewAuthMiddleware(jwtService *auth.JWTService) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
	}
}

// AuthMiddleware проверяет JWT токен
func (am *AuthMiddleware) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Исключаем публичные маршруты
		publicRoutes := []string{"/", "/health", "/api/auth/login", "/api/auth/register"}

		// Проверяем, является ли текущий путь публичным
		isPublic := false
		for _, route := range publicRoutes {
			if r.URL.Path == route {
				isPublic = true
				break
			}
		}

		if isPublic {
			next.ServeHTTP(w, r)
			return
		}

		// Извлекаем токен из заголовка
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Printf("❌ No authorization header for %s %s", r.Method, r.URL.Path)
			http.Error(w, `{"error": "Authorization header required"}`, http.StatusUnauthorized)
			return
		}

		// Проверяем формат заголовка
		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
			log.Printf("❌ Invalid authorization format for %s %s", r.Method, r.URL.Path)
			http.Error(w, `{"error": "Invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		token := bearerToken[1]

		// Валидируем токен
		claims, err := am.jwtService.ValidateToken(token)
		if err != nil {
			log.Printf("❌ Invalid token for %s %s: %v", r.Method, r.URL.Path, err)
			http.Error(w, `{"error": "Invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Добавляем claims в контекст запроса
		ctx := r.Context()
		ctx = SetUserClaims(ctx, claims)
		r = r.WithContext(ctx)

		log.Printf("✅ Authenticated user %s (role: %s) for %s %s",
			claims.Email, claims.Role, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// Вспомогательные функции для работы с контекстом
type contextKey string

const (
	userClaimsKey contextKey = "userClaims"
)

// SetUserClaims добавляет claims пользователя в контекст
func SetUserClaims(ctx context.Context, claims *auth.JWTClaims) context.Context {
	return context.WithValue(ctx, userClaimsKey, claims)
}

// GetUserClaims извлекает claims пользователя из контекста
func GetUserClaims(ctx context.Context) *auth.JWTClaims {
	if claims, ok := ctx.Value(userClaimsKey).(*auth.JWTClaims); ok {
		return claims
	}
	return nil
}
