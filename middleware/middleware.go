package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		method := c.Request.Method

		if raw != "" {
			path = path + "?" + raw
		}

		log.Printf("[%s] %d %s %s (%v)",
			method,
			statusCode,
			path,
			c.ClientIP(),
			latency,
		)
	}
}

func Recovery() gin.HandlerFunc {
	return gin.Recovery()
}
