package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kirklin/boot-backend-go-clean/pkg/logger"
)

// AccessLog returns a middleware that logs every HTTP request in structured
// JSON format, compatible with ELK, Loki, Datadog, and other log aggregators.
//
// Each log entry includes: method, path, status code, latency, client IP,
// request ID, user ID (if authenticated), user agent, and error messages.
func AccessLogMiddleware(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Build a request-scoped logger with RequestID pre-set.
		// Downstream code (controllers, usecases) can retrieve it via
		// logger.FromContext(ctx) — all their log entries will automatically
		// carry the request_id field.
		reqLog := log
		if requestID, exists := c.Get(ContextKeyRequestID); exists {
			reqLog = log.WithTrace(requestID.(string))
		}
		c.Request = c.Request.WithContext(logger.NewContext(c.Request.Context(), reqLog))

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Build structured fields
		fields := logger.Fields{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"latency_ms": float64(latency.Microseconds()) / 1000.0,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
			"body_bytes": c.Writer.Size(),
		}

		// Attach query string if present
		if rawQuery := c.Request.URL.RawQuery; rawQuery != "" {
			fields["query"] = rawQuery
		}

		// Attach request ID if set by RequestID middleware
		if requestID, exists := c.Get(ContextKeyRequestID); exists {
			fields["request_id"] = requestID
		}

		// Attach user ID if authenticated (set by JWTAuthMiddleware)
		if userID, exists := c.Get(ContextKeyUserID); exists {
			fields["user_id"] = userID
		}

		// Attach error messages if any
		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		// Choose log level based on status code
		status := c.Writer.Status()
		switch {
		case status >= 500:
			log.Log(c.Request.Context(), logger.ErrorLevel, "HTTP request", fields)
		case status >= 400:
			log.Log(c.Request.Context(), logger.WarnLevel, "HTTP request", fields)
		default:
			log.Log(c.Request.Context(), logger.InfoLevel, "HTTP request", fields)
		}
	}
}
