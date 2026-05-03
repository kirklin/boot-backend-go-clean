package route

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/gateway"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"
	"github.com/kirklin/boot-backend-go-clean/pkg/openapi"
	"github.com/kirklin/boot-backend-go-clean/pkg/version"
)

// Router holds shared dependencies that every route group needs.
// Feature-specific controllers are passed to each register method
// so that route files cannot access controllers that don't belong to them.
type Router struct {
	authenticator gateway.Authenticator
	config        *configs.AppConfig
}

// NewRouter creates a Router with shared dependencies.
func NewRouter(authenticator gateway.Authenticator, config *configs.AppConfig) *Router {
	return &Router{
		authenticator: authenticator,
		config:        config,
	}
}

// Setup registers all middleware and routes on the given engine.
func (r *Router) Setup(
	engine *gin.Engine,
	authCtrl *controller.AuthController,
	userCtrl *controller.UserController,
	infraCtrl *controller.InfraController,
) {
	// Global middleware
	if r.config.RateLimitPerMinute > 0 {
		limiter := middleware.NewRateLimiter(r.config.RateLimitPerMinute, time.Minute)
		engine.Use(limiter.LimitMiddleware())
	}
	engine.Use(middleware.CORSMiddleware())
	engine.Use(middleware.MetricsMiddleware())

	// Infrastructure routes (root-level: /, /metrics, /health/*)
	r.registerInfraRoutes(engine, infraCtrl)

	// ── OpenAPI 3.1 spec builder ──────────────────────────────────────
	spec := openapi.NewSpec("Boot Backend API", version.Version)
	spec.AddServer("/v1/api", "API Server")
	spec.AddBearerAuth("bearer")

	// Business API routes (versioned: /v1/api/*)
	api := openapi.NewAPI(engine.Group("/v1/api"), spec)
	r.registerAuthRoutes(api, authCtrl)
	r.registerUserRoutes(api, userCtrl)

	// Serve OpenAPI spec + Scalar docs (non-production only)
	if r.config.Environment != "production" {
		spec.Mount(engine)
	}
}
