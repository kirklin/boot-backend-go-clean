package route

import (
	"github.com/gin-gonic/gin"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
)

// SetupRoutes configures the routes for the application
func SetupRoutes(router *gin.Engine, db database.Database) {
	// Public routes
	publicRouter := router.Group("")
	publicRouter.GET("/health", func(c *gin.Context) {
		c.JSON(200, entity.NewSuccessResponse("success", nil))
	})

	// API routes
	apiRouter := router.Group("/api")

	// Setup auth routes
	NewAuthRouter(db, apiRouter)

	// Setup user routes
	NewUserRouter(db, apiRouter)
}
