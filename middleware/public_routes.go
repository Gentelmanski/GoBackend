package middleware

import (
	"strings"
)

// IsPublicRoute проверяет, является ли маршрут публичным
func IsPublicRoute(path string) bool {
	publicRoutes := []string{
		"/",
		"/health",
		"/api/auth/login",
		"/api/auth/register",
	}

	for _, route := range publicRoutes {
		if path == route {
			return true
		}
		// Для подпутей
		if strings.HasPrefix(path, "/api/auth/") {
			return true
		}
	}

	return false
}
