package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimitNotExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rps := 10
	middleware := RateLimitMiddleware(rps)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Send rps-1 requests rapidly, all should succeed
	for i := 0; i < rps-1; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d failed: expected status 200, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimitExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rps := 10
	middleware := RateLimitMiddleware(rps)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	successCount := 0
	rateLimitedCount := 0

	// Burst capacity is rps*2, so send more than that to trigger rate limiting
	totalRequests := (rps * 2) + 10
	for i := 0; i < totalRequests; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		} else if w.Code == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}

	// We should have some rate limited responses
	if rateLimitedCount == 0 {
		t.Errorf("Expected some rate limited responses, got none (success: %d, rate limited: %d)",
			successCount, rateLimitedCount)
	}

	// Total should match
	if successCount+rateLimitedCount != totalRequests {
		t.Errorf("Total responses don't match: success %d + rate limited %d != total %d",
			successCount, rateLimitedCount, totalRequests)
	}
}

func TestRateLimitBurstHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rps := 10
	middleware := RateLimitMiddleware(rps)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Bucket size is rps*2, so we should be able to handle burst of up to 2*rps
	burstSize := rps * 2
	successCount := 0

	for i := 0; i < burstSize; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		}
	}

	// We should successfully handle at least the burst size
	if successCount < burstSize {
		t.Errorf("Burst handling failed: expected at least %d successful requests, got %d",
			burstSize, successCount)
	}
}

func TestRateLimitRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rps := 10
	middleware := RateLimitMiddleware(rps)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Exhaust the rate limit
	for i := 0; i < rps*3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// Wait for bucket to refill (1 second should allow rps new tokens)
	time.Sleep(1 * time.Second)

	// Now requests should succeed again
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Recovery failed: expected status 200 after waiting, got %d", w.Code)
	}
}

func TestRateLimitConcurrentRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rps := 50
	middleware := RateLimitMiddleware(rps)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Simulate concurrent requests with goroutines
	// Send more than burst capacity (rps*2) to ensure rate limiting
	numGoroutines := 10
	requestsPerGoroutine := 30
	totalRequests := numGoroutines * requestsPerGoroutine

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	rateLimitedCount := 0

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < requestsPerGoroutine; i++ {
				req := httptest.NewRequest("GET", "/test", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				mu.Lock()
				if w.Code == http.StatusOK {
					successCount++
				} else if w.Code == http.StatusTooManyRequests {
					rateLimitedCount++
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Verify all requests got a response
	totalResponses := successCount + rateLimitedCount
	if totalResponses != totalRequests {
		t.Errorf("Response count mismatch: expected %d, got %d (success: %d, rate limited: %d)",
			totalRequests, totalResponses, successCount, rateLimitedCount)
	}

	// With concurrent load exceeding burst capacity, some requests should be rate limited
	if rateLimitedCount == 0 {
		t.Errorf("Expected some rate limiting under concurrent load, got none")
	}
}

func TestRateLimitDifferentRPS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		rps         int
		requests    int
		expectLimit bool
	}{
		{rps: 1, requests: 5, expectLimit: true},     // burst=2, requests=5, should limit
		{rps: 10, requests: 30, expectLimit: true},   // burst=20, requests=30, should limit
		{rps: 100, requests: 250, expectLimit: true}, // burst=200, requests=250, should limit
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("rps_%d", tc.rps), func(t *testing.T) {
			middleware := RateLimitMiddleware(tc.rps)

			router := gin.New()
			router.Use(middleware)
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			rateLimitedCount := 0

			for i := 0; i < tc.requests; i++ {
				req := httptest.NewRequest("GET", "/test", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code == http.StatusTooManyRequests {
					rateLimitedCount++
				}
			}

			if tc.expectLimit && rateLimitedCount == 0 {
				t.Errorf("Expected rate limiting for rps=%d with %d requests, got none",
					tc.rps, tc.requests)
			}
		})
	}
}
