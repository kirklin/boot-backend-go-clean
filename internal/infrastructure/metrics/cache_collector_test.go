package metrics

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kirklin/boot-backend-go-clean/pkg/cache"
)

type stubCache struct {
	cache.Cache
	stats cache.Stats
}

func (s stubCache) Stats() cache.Stats { return s.stats }

func TestCacheCollector_ExportsEveryCounter(t *testing.T) {
	collector := NewCacheCollector(stubCache{stats: cache.Stats{
		Hits:              70,
		Misses:            20,
		NegativeHits:      5,
		StaleHits:         5,
		Loads:             25,
		LoadErrors:        2,
		Refreshes:         3,
		SharedFlightWaits: 1,
	}})

	expected := strings.NewReader(`
# HELP cache_hits_total Reads served from a fresh cached value.
# TYPE cache_hits_total counter
cache_hits_total 70
# HELP cache_misses_total Reads that found nothing cached and had to call the origin.
# TYPE cache_misses_total counter
cache_misses_total 20
# HELP cache_negative_hits_total Reads answered from a tombstone, i.e. lookups for rows that do not exist and never reached the origin.
# TYPE cache_negative_hits_total counter
cache_negative_hits_total 5
# HELP cache_stale_hits_total Reads served past their logical expiry while a background refresh ran.
# TYPE cache_stale_hits_total counter
cache_stale_hits_total 5
# HELP cache_loads_total Calls into the origin loader.
# TYPE cache_loads_total counter
cache_loads_total 25
# HELP cache_load_errors_total Origin loader calls that failed.
# TYPE cache_load_errors_total counter
cache_load_errors_total 2
# HELP cache_refreshes_total Background stale-while-revalidate refreshes started.
# TYPE cache_refreshes_total counter
cache_refreshes_total 3
# HELP cache_shared_flight_waits_total Loads that waited for another replica's result instead of repeating the work.
# TYPE cache_shared_flight_waits_total counter
cache_shared_flight_waits_total 1
`)

	require.NoError(t, testutil.CollectAndCompare(collector, expected,
		"cache_hits_total",
		"cache_misses_total",
		"cache_negative_hits_total",
		"cache_stale_hits_total",
		"cache_loads_total",
		"cache_load_errors_total",
		"cache_refreshes_total",
		"cache_shared_flight_waits_total",
	))
}

func TestCacheCollector_HitRate(t *testing.T) {
	collector := NewCacheCollector(stubCache{stats: cache.Stats{
		Hits: 70, NegativeHits: 5, StaleHits: 5, Misses: 20,
	}})

	expected := strings.NewReader(`
# HELP cache_hit_rate Share of reads served without calling the origin, in [0, 1]. Cumulative since start, so use the counters above for a windowed view.
# TYPE cache_hit_rate gauge
cache_hit_rate 0.8
`)
	require.NoError(t, testutil.CollectAndCompare(collector, expected, "cache_hit_rate"))
}

func TestCacheCollector_Describe(t *testing.T) {
	collector := NewCacheCollector(cache.NewNoop())

	descriptions := make(chan *prometheus.Desc, 16)
	collector.Describe(descriptions)
	close(descriptions)

	assert.Len(t, descriptions, 9, "every metric must be described")
}

func TestCacheCollector_ReadsALiveCache(t *testing.T) {
	ctx := context.Background()
	c := cache.NewNoop()
	collector := NewCacheCollector(c)

	var got string
	_ = c.Get(ctx, "k", &got)
	_, err := cache.GetOrLoad(ctx, c, "k", time.Minute, func(context.Context) (string, error) {
		return "v", nil
	})
	require.NoError(t, err)

	expected := strings.NewReader(`
# HELP cache_misses_total Reads that found nothing cached and had to call the origin.
# TYPE cache_misses_total counter
cache_misses_total 2
`)
	assert.NoError(t, testutil.CollectAndCompare(collector, expected, "cache_misses_total"))
}
