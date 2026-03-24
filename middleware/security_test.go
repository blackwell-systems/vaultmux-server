package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders(t *testing.T) {
	// Set gin to test mode
	gin.SetMode(gin.TestMode)

	t.Run("all headers are set with correct values", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// Create middleware and execute
		handler := SecurityHeaders()
		handler(c)

		// Verify each header
		expectedHeaders := map[string]string{
			"X-Content-Type-Options":    "nosniff",
			"X-Frame-Options":           "DENY",
			"X-XSS-Protection":          "1; mode=block",
			"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
			"Content-Security-Policy":   "default-src 'none'; frame-ancestors 'none'",
		}

		for header, expectedValue := range expectedHeaders {
			actualValue := w.Header().Get(header)
			if actualValue != expectedValue {
				t.Errorf("Header %s: expected %q, got %q", header, expectedValue, actualValue)
			}
		}
	})

	t.Run("headers don't interfere with JSON responses", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := gin.New()
		r.Use(SecurityHeaders())
		r.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "test"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		// Verify status code
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Verify Content-Type is set by Gin
		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json; charset=utf-8" {
			t.Errorf("Expected Content-Type 'application/json; charset=utf-8', got %q", contentType)
		}

		// Verify security headers are still present
		if w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Error("Security headers missing after JSON response")
		}

		// Verify JSON body
		expectedBody := `{"message":"test"}`
		if w.Body.String() != expectedBody {
			t.Errorf("Expected body %q, got %q", expectedBody, w.Body.String())
		}
	})

	t.Run("headers are set before handler execution", func(t *testing.T) {
		headersPresentInHandler := false
		w := httptest.NewRecorder()
		r := gin.New()
		r.Use(SecurityHeaders())
		r.GET("/test", func(c *gin.Context) {
			// Check if headers are already set when handler executes
			if c.Writer.Header().Get("X-Content-Type-Options") == "nosniff" {
				headersPresentInHandler = true
			}
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		if !headersPresentInHandler {
			t.Error("Security headers were not set before handler execution")
		}
	})

	t.Run("headers persist through handler chain", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := gin.New()

		// Add security middleware first
		r.Use(SecurityHeaders())

		// Add another middleware that modifies response
		r.Use(func(c *gin.Context) {
			c.Header("X-Custom-Header", "custom-value")
			c.Next()
		})

		// Add final handler
		r.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		// Verify all security headers are still present
		expectedHeaders := map[string]string{
			"X-Content-Type-Options":    "nosniff",
			"X-Frame-Options":           "DENY",
			"X-XSS-Protection":          "1; mode=block",
			"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
			"Content-Security-Policy":   "default-src 'none'; frame-ancestors 'none'",
		}

		for header, expectedValue := range expectedHeaders {
			actualValue := w.Header().Get(header)
			if actualValue != expectedValue {
				t.Errorf("Header %s not persisted: expected %q, got %q", header, expectedValue, actualValue)
			}
		}

		// Verify custom header from second middleware also present
		if w.Header().Get("X-Custom-Header") != "custom-value" {
			t.Error("Custom header from second middleware not present")
		}
	})

	t.Run("headers are set with actual HTTP requests", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := gin.New()
		r.Use(SecurityHeaders())
		r.POST("/api/test", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"status": "created"})
		})

		req := httptest.NewRequest("POST", "/api/test", nil)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		// Verify all headers in actual HTTP response
		response := w.Result()
		defer response.Body.Close()

		expectedHeaders := map[string]string{
			"X-Content-Type-Options":    "nosniff",
			"X-Frame-Options":           "DENY",
			"X-XSS-Protection":          "1; mode=block",
			"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
			"Content-Security-Policy":   "default-src 'none'; frame-ancestors 'none'",
		}

		for header, expectedValue := range expectedHeaders {
			actualValue := response.Header.Get(header)
			if actualValue != expectedValue {
				t.Errorf("Header %s in HTTP response: expected %q, got %q", header, expectedValue, actualValue)
			}
		}
	})
}
