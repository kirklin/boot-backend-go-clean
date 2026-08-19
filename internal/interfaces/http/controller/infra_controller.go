package controller

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity/response"
	"github.com/kirklin/boot-backend-go-clean/pkg/cache"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
	"github.com/kirklin/boot-backend-go-clean/pkg/storage"
	"github.com/kirklin/boot-backend-go-clean/pkg/version"
)

// probeTimeout bounds each dependency check, so a hung dependency reports as
// down instead of holding the readiness lock open.
const probeTimeout = 2 * time.Second

// InfraController handles infrastructure endpoints: welcome page, health probes.
type InfraController struct {
	db      database.Database
	cache   cache.Cache
	storage storage.Storage
	config  *configs.AppConfig

	// Caching to prevent DB DoS via healthcheck endpoint
	mu               sync.RWMutex
	lastCheckTime    time.Time
	cachedAllHealthy bool
	cachedChecks     gin.H
	cacheTTL         time.Duration
}

// NewInfraController creates a new InfraController. store may be nil when
// object storage is disabled.
func NewInfraController(
	db database.Database,
	cacheClient cache.Cache,
	store storage.Storage,
	config *configs.AppConfig,
) *InfraController {
	return &InfraController{
		db:       db,
		cache:    cacheClient,
		storage:  store,
		config:   config,
		cacheTTL: 5 * time.Second,
	}
}

// Welcome returns basic application info.
func (h *InfraController) Welcome(c *gin.Context) {
	data := gin.H{
		"version": version.Version,
	}
	if h.config.Environment != "production" {
		data["git_commit"] = version.GitCommit
		data["build_time"] = version.BuildTime
	}
	c.JSON(http.StatusOK, response.NewSuccessResponse("Boot Backend Go Clean is running", data))
}

// Live is a liveness probe. It returns 200 if the process is running.
// Use this for Kubernetes livenessProbe or Docker HEALTHCHECK.
//
// A failing liveness probe means the process is deadlocked or unrecoverable,
// and the container should be restarted.
func (h *InfraController) Live(c *gin.Context) {
	c.JSON(http.StatusOK, response.NewSuccessResponse("alive", gin.H{
		"version": version.Version,
	}))
}

// Ready is a readiness probe: 200 only if the database, and any enabled cache
// and object storage, are reachable. Use it for Kubernetes readinessProbe.
func (h *InfraController) Ready(c *gin.Context) {
	h.mu.RLock()

	if time.Since(h.lastCheckTime) < h.cacheTTL {
		isHealthy := h.cachedAllHealthy
		checks := h.cachedChecks
		h.mu.RUnlock()
		h.respond(c, isHealthy, checks)
		return
	}
	h.mu.RUnlock()

	// Cache expired, acquire write lock to run the probes
	h.mu.Lock()
	defer h.mu.Unlock()

	// Double-check pattern (another goroutine might have refreshed it while we waited for lock)
	if time.Since(h.lastCheckTime) < h.cacheTTL {
		h.respond(c, h.cachedAllHealthy, h.cachedChecks)
		return
	}

	checks, allHealthy := h.runChecks(c.Request.Context())

	// Update cache
	h.cachedAllHealthy = allHealthy
	h.cachedChecks = checks
	h.lastCheckTime = time.Now()

	h.respond(c, allHealthy, checks)
}

func (h *InfraController) runChecks(ctx context.Context) (gin.H, bool) {
	checks := gin.H{}
	allHealthy := true

	record := func(name string, probe func(context.Context) error) {
		probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
		defer cancel()

		if err := probe(probeCtx); err != nil {
			checks[name] = gin.H{"status": "down", "error": h.formatErr(err)}
			allHealthy = false
			return
		}
		checks[name] = gin.H{"status": "up"}
	}

	record("database", func(probeCtx context.Context) error {
		sqlDB, err := h.db.DB().DB()
		if err != nil {
			return err
		}
		return sqlDB.PingContext(probeCtx)
	})

	if h.config.CacheEnabled && h.cache != nil {
		record("cache", h.cache.Ping)
	}

	if h.storage != nil {
		record("storage", h.storage.Ping)
	}

	return checks, allHealthy
}

// Helper to format errors based on environment
func (h *InfraController) formatErr(err error) string {
	if h.config.Environment == "production" {
		return "service unavailable"
	}
	return err.Error()
}

func (h *InfraController) respond(c *gin.Context, healthy bool, checks gin.H) {
	if !healthy {
		c.JSON(http.StatusServiceUnavailable, response.NewSuccessResponse("not ready", checks))
		return
	}
	c.JSON(http.StatusOK, response.NewSuccessResponse("ready", checks))
}
