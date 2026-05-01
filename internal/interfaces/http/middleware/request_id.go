package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
	snowflakeutils "github.com/kirklin/boot-backend-go-clean/pkg/utils/snowflake"
)

// RequestID middleware generates a unique request ID for each incoming request using Snowflake ID
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// Using Snowflake ID instead of UUID for better performance, smaller size,
			// and time-ordered sortability in logs.
			requestID = strconv.FormatInt(snowflakeutils.NextID(), 10)
		}

		// Put it into the Gin context
		c.Set("x-request-id", requestID)

		// Also attach it to the response header
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}
