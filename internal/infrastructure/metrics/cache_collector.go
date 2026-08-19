// Package metrics adapts infrastructure counters to Prometheus.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/kirklin/boot-backend-go-clean/pkg/cache"
)

// CacheCollector exports cache.Stats to Prometheus, reading on scrape.
type CacheCollector struct {
	source cache.Cache

	hits              *prometheus.Desc
	misses            *prometheus.Desc
	negativeHits      *prometheus.Desc
	staleHits         *prometheus.Desc
	loads             *prometheus.Desc
	loadErrors        *prometheus.Desc
	refreshes         *prometheus.Desc
	sharedFlightWaits *prometheus.Desc
	hitRate           *prometheus.Desc
}

// NewCacheCollector builds a collector reading from source.
func NewCacheCollector(source cache.Cache) *CacheCollector {
	return &CacheCollector{
		source: source,
		hits: prometheus.NewDesc("cache_hits_total",
			"Reads served from a fresh cached value.", nil, nil),
		misses: prometheus.NewDesc("cache_misses_total",
			"Reads that found nothing cached and had to call the origin.", nil, nil),
		negativeHits: prometheus.NewDesc("cache_negative_hits_total",
			"Reads answered from a tombstone, i.e. lookups for rows that do not exist and never reached the origin.", nil, nil),
		staleHits: prometheus.NewDesc("cache_stale_hits_total",
			"Reads served past their logical expiry while a background refresh ran.", nil, nil),
		loads: prometheus.NewDesc("cache_loads_total",
			"Calls into the origin loader.", nil, nil),
		loadErrors: prometheus.NewDesc("cache_load_errors_total",
			"Origin loader calls that failed.", nil, nil),
		refreshes: prometheus.NewDesc("cache_refreshes_total",
			"Background stale-while-revalidate refreshes started.", nil, nil),
		sharedFlightWaits: prometheus.NewDesc("cache_shared_flight_waits_total",
			"Loads that waited for another replica's result instead of repeating the work.", nil, nil),
		hitRate: prometheus.NewDesc("cache_hit_rate",
			"Share of reads served without calling the origin, in [0, 1]. Cumulative since start, so use the counters above for a windowed view.", nil, nil),
	}
}

// Describe implements prometheus.Collector.
func (c *CacheCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.hits
	ch <- c.misses
	ch <- c.negativeHits
	ch <- c.staleHits
	ch <- c.loads
	ch <- c.loadErrors
	ch <- c.refreshes
	ch <- c.sharedFlightWaits
	ch <- c.hitRate
}

// Collect implements prometheus.Collector.
func (c *CacheCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.source.Stats()

	counter := func(desc *prometheus.Desc, value uint64) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, float64(value))
	}

	counter(c.hits, stats.Hits)
	counter(c.misses, stats.Misses)
	counter(c.negativeHits, stats.NegativeHits)
	counter(c.staleHits, stats.StaleHits)
	counter(c.loads, stats.Loads)
	counter(c.loadErrors, stats.LoadErrors)
	counter(c.refreshes, stats.Refreshes)
	counter(c.sharedFlightWaits, stats.SharedFlightWaits)

	ch <- prometheus.MustNewConstMetric(c.hitRate, prometheus.GaugeValue, stats.HitRate())
}
