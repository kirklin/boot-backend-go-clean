package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/pkg/jwt"
)

// JWTAuthMiddleware checks for a valid JWT token in the Authorization header
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, entity.NewErrorResponse("Authorization header is required", nil))
			c.Abort()
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || strings.ToLower(bearerToken[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, entity.NewErrorResponse("Invalid authorization header format", nil))
			c.Abort()
			return
		}

		tokenString := bearerToken[1]

		// Validate the access token
		claims, _, err := jwt.ValidateAccessToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, entity.NewErrorResponse("Invalid or expired token", err))
			c.Abort()
			return
		}

		// Set the user information in the context
		c.Set("x-user-id", claims.UserID)
		c.Set("x-username", claims.Username)

		c.Next()
	}
}
