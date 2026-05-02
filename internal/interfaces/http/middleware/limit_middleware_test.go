package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRateLimitedRouter(limit int, window time.Duration) *gin.Engine {
	r := gin.New()
	rl := &RateLimiter{
		limit:          limit,
		windowDuration: window,
		requests:       make(map[string]int),
		resetTime:      make(map[string]time.Time),
	}
	r.Use(rl.LimitMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	router := setupRateLimitedRouter(3, 1*time.Minute)

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	router := setupRateLimitedRouter(2, 1*time.Minute)

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Third request should be rate-limited
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "Rate limit exceeded")
}

func TestRateLimiter_SetsHeaders(t *testing.T) {
	router := setupRateLimitedRouter(5, 1*time.Minute)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, "5", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "4", w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
}

func TestRateLimiter_ResetsAfterWindow(t *testing.T) {
	router := setupRateLimitedRouter(1, 50*time.Millisecond)

	// First request should succeed
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Second request should be blocked
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Wait for window to reset
	time.Sleep(60 * time.Millisecond)

	// Third request should succeed after window reset
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := &RateLimiter{
		limit:          5,
		windowDuration: 1 * time.Minute,
		requests:       make(map[string]int),
		resetTime:      make(map[string]time.Time),
	}

	// Add expired entries
	rl.requests["1.2.3.4"] = 5
	rl.resetTime["1.2.3.4"] = time.Now().Add(-1 * time.Minute)

	rl.requests["5.6.7.8"] = 2
	rl.resetTime["5.6.7.8"] = time.Now().Add(1 * time.Minute)

	rl.cleanup()

	// Expired entry should be removed
	assert.NotContains(t, rl.requests, "1.2.3.4")
	// Active entry should remain
	assert.Contains(t, rl.requests, "5.6.7.8")
}
