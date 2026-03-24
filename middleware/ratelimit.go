package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimitMiddleware creates a Gin middleware that limits requests per second.
// Uses golang.org/x/time/rate for token bucket rate limiting.
// Returns HTTP 429 (Too Many Requests) when rate limit is exceeded.
func RateLimitMiddleware(rps int) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(rps), rps*2)
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	}
}
