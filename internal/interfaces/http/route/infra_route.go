package route

import (
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
)

// registerInfraRoutes registers infrastructure endpoints at the root level.
// These are NOT business APIs and should NOT be under versioned paths.
func (r *Router) registerInfraRoutes(engine *gin.Engine, ctrl *controller.InfraController) {
	// Welcome
	engine.GET("/", ctrl.Welcome)

	// Prometheus metrics
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// pprof profiling (non-production only)
	if r.config.Environment != "production" {
		pprof.Register(engine)
	}

	// Health probes
	health := engine.Group("/health")
	{
		health.GET("", ctrl.Live)       // backward compat: /health
		health.GET("/live", ctrl.Live)   // liveness probe
		health.GET("/ready", ctrl.Ready) // readiness probe
	}
}
