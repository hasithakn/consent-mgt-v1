package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CorrelationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := extractCorrelationID(c)
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		c.Set("correlation_id", correlationID)
		c.Header("X-Correlation-ID", correlationID)
		c.Next()
	}
}

func extractCorrelationID(c *gin.Context) string {
	headers := []string{"X-Correlation-ID", "X-Request-ID", "X-Trace-ID"}
	for _, header := range headers {
		if id := c.GetHeader(header); id != "" {
			return id
		}
	}
	return ""
}

// WrapWithCorrelationID wraps an http.Handler with correlation ID middleware
func WrapWithCorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract or generate correlation ID
		correlationID := extractCorrelationIDFromRequest(r)
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Set correlation ID in response header
		w.Header().Set("X-Correlation-ID", correlationID)

		// Add correlation ID to request context
		ctx := context.WithValue(r.Context(), "correlation_id", correlationID)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractCorrelationIDFromRequest(r *http.Request) string {
	headers := []string{"X-Correlation-ID", "X-Request-ID", "X-Trace-ID"}
	for _, header := range headers {
		if id := r.Header.Get(header); id != "" {
			return id
		}
	}
	return ""
}
