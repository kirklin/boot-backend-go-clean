package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"

	snowflakeutils "github.com/kirklin/boot-backend-go-clean/pkg/utils/snowflake"
)

// RequestIDMiddleware generates a unique request ID for each incoming request using Snowflake ID.
// If the client sends an X-Request-ID header (e.g. from a reverse proxy), it is reused;
// otherwise a new Snowflake ID is generated for performance and time-ordered sortability.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(HeaderRequestID)
		if requestID == "" {
			requestID = strconv.FormatInt(snowflakeutils.NextID(), 10)
		}

		c.Set(ContextKeyRequestID, requestID)
		c.Header(HeaderRequestID, requestID)

		c.Next()
	}
}
