package route

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity/response"
	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/auth"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"

	"github.com/kirklin/boot-backend-go-clean/pkg/database"
)

// SetupRoutes configures the routes for the application
func SetupRoutes(router *gin.Engine, db database.Database, config *configs.AppConfig) {

	if config.RateLimitPerMinute > 0 {
		limiter := middleware.NewRateLimiter(config.RateLimitPerMinute, time.Minute)
		router.Use(limiter.LimitMiddleware())
	}

	router.Use(middleware.CORSMiddleware())

	// Root route
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, response.NewSuccessResponse("Boot Backend Go Clean is running", gin.H{
			"version": "1.0.0",
		}))
	})

	// API routes
	apiRouter := router.Group("/v1/api")

	// Health check under API group
	apiRouter.GET("/health", func(c *gin.Context) {
		c.JSON(200, response.NewSuccessResponse[any]("success", nil))
	})

	tokenBlacklist := auth.NewTokenBlacklist()
	authenticator := auth.NewJWTAuthenticator(
		config.AccessTokenSecret,
		config.RefreshTokenSecret,
		config.JWTIssuer,
		time.Duration(config.AccessTokenLifetime)*time.Hour,
		time.Duration(config.RefreshTokenLifetime)*time.Hour,
		tokenBlacklist,
	)

	// Setup auth routes
	NewAuthRouter(db, apiRouter, config, authenticator)

	// Setup user routes
	NewUserRouter(db, apiRouter, config, authenticator)
}
