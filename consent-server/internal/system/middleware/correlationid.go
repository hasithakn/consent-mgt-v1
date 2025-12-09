package middleware

import (
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
