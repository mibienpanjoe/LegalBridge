package api

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// allowedOrigins is the fixed set of CORS origins accepted by the API.
// Production Vercel origin is added in Phase 8.
var allowedOrigins = map[string]bool{
	"http://localhost:3000": true,
}

// CORSMiddleware sets CORS headers for allowed origins and handles preflight
// OPTIONS requests.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if allowedOrigins[origin] {
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
