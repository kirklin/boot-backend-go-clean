package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// reqCount is a counter that tracks the total number of HTTP requests
	reqCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed, partitioned by status code, method, and HTTP path.",
		},
		[]string{"status", "method", "path"},
	)

	// reqDuration is a histogram that tracks the latency of HTTP requests
	reqDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Latency of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets, // default buckets: 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
		},
		[]string{"method", "path"},
	)
)

// MetricsMiddleware returns a Gin middleware that records Prometheus metrics
// for each incoming HTTP request.
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start).Seconds()

		// Extract metrics labels
		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		path := c.FullPath()

		// If path is empty, it means the router didn't find a matching route (e.g. 404).
		// We use the raw path in this case, but be careful of high cardinality if
		// users request random paths. It's safer to just group them under "UNKNOWN" or similar,
		// but using the raw path helps identify what's being hit. We'll use "UNKNOWN_ROUTE"
		// to avoid infinite cardinality from malicious bots hitting random URLs.
		if path == "" {
			path = "UNKNOWN_ROUTE"
		}

		// Record metrics
		reqCount.WithLabelValues(status, method, path).Inc()
		reqDuration.WithLabelValues(method, path).Observe(latency)
	}
}
