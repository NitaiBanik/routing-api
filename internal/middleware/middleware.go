package middleware

import (
	"net/http"

	"go.uber.org/zap"
)

func LoggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			zap.L().Info("ROUTING-API",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
			)
		})
	}
}
