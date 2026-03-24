package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders creates a Gin middleware that sets standard security headers
// to protect against XSS, clickjacking, MIME sniffing, and protocol downgrade attacks.
//
// Headers set:
//   - X-Content-Type-Options: nosniff - Prevents browsers from MIME-type sniffing
//   - X-Frame-Options: DENY - Prevents page from being embedded in iframe (clickjacking protection)
//   - X-XSS-Protection: 1; mode=block - Enables browser XSS filter (legacy browser support)
//   - Strict-Transport-Security: max-age=31536000; includeSubDomains - Enforces HTTPS for 1 year
//   - Content-Security-Policy: default-src 'none'; frame-ancestors 'none' - Restricts all resource loading (API doesn't serve HTML/JS)
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// XSS protection (legacy browsers)
		c.Header("X-XSS-Protection", "1; mode=block")

		// Force HTTPS (when deployed with TLS)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Content Security Policy (API-only, no resources)
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		c.Next()
	}
}
