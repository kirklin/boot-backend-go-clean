package middleware

import (
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity/response"
)

type RateLimiter struct {
	mu             sync.Mutex
	limit          int           // 每个窗口的最大请求数
	windowDuration time.Duration // 时间窗口，单位：秒
	requests       map[string]int
	resetTime      map[string]time.Time
}

// NewRateLimiter 构建限流器，传入每个 IP 的限制请求次数和时间窗口（秒）
func NewRateLimiter(limit int, duration time.Duration) *RateLimiter {
	rl := &RateLimiter{
		limit:          limit,
		windowDuration: duration,
		requests:       make(map[string]int),
		resetTime:      make(map[string]time.Time),
	}

	// 启动后台清理任务，防止由于不断新增 IP 导致内存泄漏
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	for ip, expireTime := range rl.resetTime {
		if now.After(expireTime) {
			delete(rl.requests, ip)
			delete(rl.resetTime, ip)
		}
	}
}

// LimitMiddleware 实现 Gin 中间件
func (rl *RateLimiter) LimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		rl.mu.Lock()
		defer rl.mu.Unlock()

		count, exists := rl.requests[clientIP]
		resetTime, resetExists := rl.resetTime[clientIP]

		// 如果超过时间窗口，重置计数
		if !resetExists || time.Now().After(resetTime) {
			rl.requests[clientIP] = 0
			count = 0
			rl.resetTime[clientIP] = time.Now().Add(rl.windowDuration)
			resetTime = rl.resetTime[clientIP] // 修复: 更新局部变量，防止返回零时间戳
		}

		// 判断是否超过限制
		if exists && count >= rl.limit {
			c.JSON(http.StatusTooManyRequests, response.NewErrorResponse(
				"Rate limit exceeded. Please try again later.", errors.New("too many requests"),
			))
			c.Abort()
			return
		}

		// 增加计数
		rl.requests[clientIP]++
		remaining := rl.limit - (count + 1)

		c.Header("X-RateLimit-Limit", strconv.Itoa(rl.limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", resetTime.Format(time.RFC3339))

		c.Next()
	}
}
