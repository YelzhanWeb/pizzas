package http

import (
	"fmt"
	"net/http"
	"time"
	"wheres-my-pizza/internal/adapter/logger"
)

func LoggingMiddleware(logger logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())

			logger.Debug("http_request", fmt.Sprintf("%s %s", r.Method, r.URL.Path), requestID, map[string]interface{}{
				"method": r.Method,
				"path":   r.URL.Path,
			})

			next.ServeHTTP(w, r)

			duration := time.Since(start)
			logger.Debug("http_response", "Request completed", requestID, map[string]interface{}{
				"duration_ms": duration.Milliseconds(),
			})
		})
	}
}

func RecoveryMiddleware(logger logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())
					logger.Error("panic_recovered", "Panic recovered", requestID, nil, fmt.Errorf("%v", err))
					http.Error(w, "Internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
