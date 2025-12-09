package middleware

import (
	"github.com/gin-gonic/gin"
)

type CORSOptions struct {
	AllowedOrigins   []string
	AllowedMethods   string
	AllowedHeaders   string
	AllowCredentials bool
}

func CORSMiddleware(opts CORSOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && isOriginAllowed(origin, opts.AllowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", opts.AllowedMethods)
			c.Header("Access-Control-Allow-Headers", opts.AllowedHeaders)
			if opts.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}
		}
		c.Next()
	}
}

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}
