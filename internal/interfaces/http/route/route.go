package route

import (
	"github.com/gin-gonic/gin"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity/response"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"
	"time"

	"github.com/kirklin/boot-backend-go-clean/pkg/database"
)

// SetupRoutes configures the routes for the application
func SetupRoutes(router *gin.Engine, db database.Database, config *configs.AppConfig) {

	limiter := middleware.NewRateLimiter(200, time.Minute)
	router.Use(limiter.LimitMiddleware())

	router.Use(middleware.CORSMiddleware())

	// Public routes
	publicRouter := router.Group("")
	publicRouter.GET("/health", func(c *gin.Context) {
		c.JSON(200, response.NewSuccessResponse[any]("success", nil))
	})

	// API routes
	apiRouter := router.Group("/v1/api")

	// Setup auth routes
	NewAuthRouter(db, apiRouter, config)

	// Setup user routes
	NewUserRouter(db, apiRouter, config)
}
