package route

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity/response"
	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/auth"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
	"github.com/kirklin/boot-backend-go-clean/pkg/version"
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
		data := gin.H{
			"version": version.Version,
		}
		// Only expose build details in non-production environments
		if config.Environment != "production" {
			data["git_commit"] = version.GitCommit
			data["build_time"] = version.BuildTime
		}
		c.JSON(200, response.NewSuccessResponse("Boot Backend Go Clean is running", data))
	})

	// API routes
	apiRouter := router.Group("/v1/api")

	// Health check endpoints
	healthCtrl := controller.NewHealthController(db, config)
	healthGroup := apiRouter.Group("/health")
	{
		healthGroup.GET("", healthCtrl.Live)        // backward compat: /v1/api/health
		healthGroup.GET("/live", healthCtrl.Live)    // liveness probe: process is running
		healthGroup.GET("/ready", healthCtrl.Ready)  // readiness probe: DB is reachable
	}

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
