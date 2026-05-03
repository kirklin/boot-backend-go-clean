package route

import (
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/pkg/openapi"
)

// registerInfraRoutes registers infrastructure endpoints at the root level.
// These are NOT business APIs and should NOT be under versioned paths.
// They use RawGet because they manage their own response format.
func (r *Router) registerInfraRoutes(engine *gin.Engine, infraAPI *openapi.API, ctrl *controller.InfraController) {
	// Welcome
	openapi.RawGet(infraAPI, "/", ctrl.Welcome,
		openapi.Summary("服务欢迎页"),
		openapi.Tags("Infrastructure"),
	)

	// Prometheus metrics — registered directly (not in OpenAPI spec)
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// pprof profiling (non-production only)
	if r.config.Environment != "production" {
		pprof.Register(engine)
	}

	// Health probes
	healthAPI := infraAPI.Group("/health")

	openapi.RawGet(healthAPI, "", ctrl.Live,
		openapi.Summary("健康检查（向后兼容）"),
		openapi.Tags("Infrastructure"),
	)

	openapi.RawGet(healthAPI, "/live", ctrl.Live,
		openapi.Summary("存活探针"),
		openapi.Description("Kubernetes livenessProbe — 进程是否存活"),
		openapi.Tags("Infrastructure"),
	)

	openapi.RawGet(healthAPI, "/ready", ctrl.Ready,
		openapi.Summary("就绪探针"),
		openapi.Description("Kubernetes readinessProbe — 依赖（数据库等）是否可用"),
		openapi.Tags("Infrastructure"),
	)
}
