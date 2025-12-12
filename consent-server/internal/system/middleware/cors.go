package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type CORSOptions struct {
	AllowedOrigins   []string
	AllowedMethods   string
	AllowedHeaders   string
	AllowCredentials bool
	// For http.ServeMux
	AllowOrigin  string
	AllowMethods []string
	AllowHeaders []string
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

// WithCORS wraps an http.HandlerFunc with CORS headers for http.ServeMux
func WithCORS(pattern string, handler http.HandlerFunc, opts CORSOptions) (string, http.HandlerFunc) {
	wrapped := func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		origin := r.Header.Get("Origin")
		if origin == "" || opts.AllowOrigin == "*" {
			// w.Header().Set("Access-Control-Allow-Origin", opts.AllowOrigin)
		} else {
			// w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		// w.Header().Set("Access-Control-Allow-Methods", strings.Join(opts.AllowMethods, ", "))
		// w.Header().Set("Access-Control-Allow-Headers", strings.Join(opts.AllowHeaders, ", "))
		if opts.AllowCredentials {
			// w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			// w.WriteHeader(http.StatusNoContent)
			return
		}

		// Call the actual handler
		handler(w, r)
	}
	return pattern, wrapped
}

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}
