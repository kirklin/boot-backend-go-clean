package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity/response"
)

// ErrorHandler is a global middleware that catches uncaught errors from c.Error() and formats them
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if any errors occurred during the request lifecycle
		if len(c.Errors) > 0 {
			// Do not write if response has already been written by a controller
			if c.Writer.Written() {
				return
			}

			// Extract the last error
			ginErr := c.Errors.Last()
			actualErr := ginErr.Err

			// For simplicity, we just wrap it into our standard response format.
			// In production, we hide the actual error message to prevent sensitive information leakage (like SQL errors).
			errMessage := actualErr.Error()
			if gin.Mode() == gin.ReleaseMode {
				errMessage = "Internal Server Error"
			}

			c.JSON(http.StatusInternalServerError, response.NewErrorResponse(errMessage, nil))
			c.Abort()
		}
	}
}
