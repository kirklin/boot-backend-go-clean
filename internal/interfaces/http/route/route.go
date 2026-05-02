package route

import (
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity/response"
	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/auth"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/humaerr"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
	"github.com/kirklin/boot-backend-go-clean/pkg/version"
)

// SetupRoutes configures the routes for the application
func SetupRoutes(router *gin.Engine, db database.Database, config *configs.AppConfig) {
	// Install custom huma.NewError to map domain AppError to correct HTTP status
	humaerr.Setup()

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

	// Root route (still using gin directly — not part of the API spec)
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

	// Create a Gin router group for the API
	apiGroup := router.Group("/v1/api")

	// Create the Huma API instance using the API group
	humaConfig := huma.DefaultConfig("Boot Backend Go Clean API", version.Version)
	humaConfig.Info.Description = "A RESTful API built with Go, following Clean Architecture principles."
	humaConfig.Info.Contact = &huma.Contact{
		Name: "Kirk Lin",
		URL:  "https://github.com/kirklin",
	}

	// Add security scheme for Bearer JWT
	humaConfig.Components = &huma.Components{
		SecuritySchemes: map[string]*huma.SecurityScheme{
			"bearerAuth": {
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
				Description:  "JWT Bearer token authentication",
			},
		},
	}

	// Set the API base path so that Huma's built-in docs page generates the
	// correct OpenAPI spec URL (e.g. /v1/api/openapi.json instead of /openapi.json)
	humaConfig.Servers = []*huma.Server{
		{URL: "/v1/api", Description: "API v1"},
	}

	// Use Huma's built-in Scalar docs renderer
	humaConfig.DocsRenderer = huma.DocsRendererScalar

	api := humagin.NewWithGroup(router, apiGroup, humaConfig)

	// Health check endpoints (public, no auth)
	healthCtrl := controller.NewHealthController(db, config)
	huma.Register(api, huma.Operation{
		OperationID: "health-live",
		Method:      http.MethodGet,
		Path:        "/v1/api/health",
		Summary:     "Liveness probe (backward compat)",
		Tags:        []string{"Health"},
	}, healthCtrl.Live)

	huma.Register(api, huma.Operation{
		OperationID: "health-live-explicit",
		Method:      http.MethodGet,
		Path:        "/v1/api/health/live",
		Summary:     "Liveness probe: process is running",
		Tags:        []string{"Health"},
	}, healthCtrl.Live)

	huma.Register(api, huma.Operation{
		OperationID: "health-ready",
		Method:      http.MethodGet,
		Path:        "/v1/api/health/ready",
		Summary:     "Readiness probe: DB is reachable",
		Tags:        []string{"Health"},
	}, healthCtrl.Ready)

	// Auth
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
	NewAuthRouter(db, api, config, authenticator)

	// Setup user routes
	NewUserRouter(db, api, router, config, authenticator)
}
