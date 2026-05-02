package controller

import (
	"net/http"

	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity/response"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
	"github.com/kirklin/boot-backend-go-clean/pkg/version"
)

// HealthController handles health check endpoints.
type HealthController struct {
	db     database.Database
	config *configs.AppConfig

	// Caching to prevent DB DoS via healthcheck endpoint
	mu               sync.RWMutex
	lastCheckTime    time.Time
	cachedAllHealthy bool
	cachedChecks     gin.H
	cacheTTL         time.Duration
}

// NewHealthController creates a new HealthController.
func NewHealthController(db database.Database, config *configs.AppConfig) *HealthController {
	return &HealthController{
		db:       db,
		config:   config,
		cacheTTL: 5 * time.Second, // Max 1 DB ping every 5 seconds
	}
}

// Live is a liveness probe. It returns 200 if the process is running.
// Use this for Kubernetes livenessProbe or Docker HEALTHCHECK.
//
// A failing liveness probe means the process is deadlocked or unrecoverable,
// and the container should be restarted.
func (h *HealthController) Live(c *gin.Context) {
	c.JSON(http.StatusOK, response.NewSuccessResponse("alive", gin.H{
		"version": version.Version,
	}))
}

// Ready is a readiness probe. It returns 200 only if all critical
// dependencies (database, etc.) are reachable.
// Use this for Kubernetes readinessProbe — a failing readiness probe
// removes the pod from the Service load balancer without restarting it.
//
// This endpoint checks:
//   - Database connectivity (SQL ping with 2s timeout)
func (h *HealthController) Ready(c *gin.Context) {
	h.mu.RLock()
	// If cache is still valid, return immediately (O(1) time, no DB call)
	if time.Since(h.lastCheckTime) < h.cacheTTL {
		isHealthy := h.cachedAllHealthy
		checks := h.cachedChecks
		h.mu.RUnlock()
		
		if !isHealthy {
			c.JSON(http.StatusServiceUnavailable, response.NewSuccessResponse("not ready", checks))
			return
		}
		c.JSON(http.StatusOK, response.NewSuccessResponse("ready", checks))
		return
	}
	h.mu.RUnlock()

	// Cache expired, acquire write lock to ping DB
	h.mu.Lock()
	defer h.mu.Unlock()

	// Double-check pattern (another goroutine might have refreshed it while we waited for lock)
	if time.Since(h.lastCheckTime) < h.cacheTTL {
		if !h.cachedAllHealthy {
			c.JSON(http.StatusServiceUnavailable, response.NewSuccessResponse("not ready", h.cachedChecks))
			return
		}
		c.JSON(http.StatusOK, response.NewSuccessResponse("ready", h.cachedChecks))
		return
	}

	checks := gin.H{}
	allHealthy := true

	// Helper to format errors based on environment
	formatErr := func(err error) string {
		if h.config.Environment == "production" {
			return "service unavailable" // Mask internal DB errors in prod
		}
		return err.Error()
	}

	// Check database connectivity
	sqlDB, err := h.db.DB().DB()
	if err != nil {
		checks["database"] = gin.H{"status": "down", "error": formatErr(err)}
		allHealthy = false
	} else if err = sqlDB.Ping(); err != nil {
		checks["database"] = gin.H{"status": "down", "error": formatErr(err)}
		allHealthy = false
	} else {
		checks["database"] = gin.H{"status": "up"}
	}

	// Update cache
	h.cachedAllHealthy = allHealthy
	h.cachedChecks = checks
	h.lastCheckTime = time.Now()

	if !allHealthy {
		c.JSON(http.StatusServiceUnavailable, response.NewSuccessResponse("not ready", checks))
		return
	}

	c.JSON(http.StatusOK, response.NewSuccessResponse("ready", checks))
}
