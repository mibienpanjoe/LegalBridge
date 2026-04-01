package api

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware sets CORS headers for allowed origins and handles preflight
// OPTIONS requests. localhost:3000 is always included; pass the production
// Vercel origin via additionalOrigins (from CORS_ALLOWED_ORIGIN env var).
func CORSMiddleware(additionalOrigins ...string) gin.HandlerFunc {
	allowed := map[string]bool{
		"http://localhost:3000": true,
	}
	for _, o := range additionalOrigins {
		if o != "" {
			allowed[o] = true
		}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if allowed[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// LoggingMiddleware logs each request method, path, status code, and latency.
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("%s %s → %d (%s)",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			time.Since(start),
		)
	}
}
