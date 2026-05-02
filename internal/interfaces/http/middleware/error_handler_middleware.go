package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity/response"
	domainerrors "github.com/kirklin/boot-backend-go-clean/internal/domain/errors"
)

// ErrorHandlerMiddleware is a middleware that handles errors set via c.Error() and formats them
func ErrorHandlerMiddleware() gin.HandlerFunc {
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

			// If it's an AppError, use the structured code, message, and HTTP status.
			var appErr *domainerrors.AppError
			if errors.As(actualErr, &appErr) {
				c.JSON(appErr.HTTPCode, response.NewErrorResponse(appErr.Message, appErr))
				c.Abort()
				return
			}

			// For unknown errors, hide details in production to prevent
			// leaking internal information (e.g. SQL errors, stack traces).
			errMessage := actualErr.Error()
			if gin.Mode() == gin.ReleaseMode {
				errMessage = "Internal Server Error"
			}

			c.JSON(http.StatusInternalServerError, response.NewErrorResponse(errMessage, nil))
			c.Abort()
		}
	}
}
