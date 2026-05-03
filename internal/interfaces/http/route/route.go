package route

import (
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity/response"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/gateway"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"
	"github.com/kirklin/boot-backend-go-clean/pkg/version"
)

// SetupRoutes configures the routes for the application.
// All controllers and the authenticator are pre-built by the Composition Root (app.Initialize)
// and passed in — this function only wires them to HTTP paths.
func SetupRoutes(
	router *gin.Engine,
	authCtrl *controller.AuthController,
	userCtrl *controller.UserController,
	healthCtrl *controller.HealthController,
	authenticator gateway.Authenticator,
	config *configs.AppConfig,
) {
	if config.RateLimitPerMinute > 0 {
		limiter := middleware.NewRateLimiter(config.RateLimitPerMinute, time.Minute)
		router.Use(limiter.LimitMiddleware())
	}

	router.Use(middleware.CORSMiddleware())

	// Record Prometheus metrics for all incoming requests
	router.Use(middleware.MetricsMiddleware())

	// Expose Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Mount pprof performance profiling endpoints under /debug/pprof/
	// (Only exposed in non-production environments for security)
	if config.Environment != "production" {
		pprof.Register(router)
	}

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
	healthGroup := apiRouter.Group("/health")
	{
		healthGroup.GET("", healthCtrl.Live)       // backward compat: /v1/api/health
		healthGroup.GET("/live", healthCtrl.Live)   // liveness probe: process is running
		healthGroup.GET("/ready", healthCtrl.Ready) // readiness probe: DB is reachable
	}

	// Setup auth routes
	NewAuthRouter(apiRouter, authCtrl, authenticator)

	// Setup user routes
	NewUserRouter(apiRouter, userCtrl, authenticator)
}
