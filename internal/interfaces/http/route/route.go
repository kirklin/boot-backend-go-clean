package route

import (
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures the routes for the application
func SetupRoutes(router *gin.Engine, db database.Database) {
	publicRouter := router.Group("")
	// Health check route
	publicRouter.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, entity.NewSuccessResponse("success", nil))
	})
	// protected routes
	protectedRouter := router.Group("/api")
	protectedRouter.Use(middleware.JWTAuthMiddleware())
	{
		// your protected routes here
		protectedRouter.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, entity.NewSuccessResponse("success", nil))
		})
	}
}
