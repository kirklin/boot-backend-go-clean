package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeClock struct{ nanos atomic.Int64 }

func newFakeClock() *fakeClock {
	clock := &fakeClock{}
	clock.nanos.Store(time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC).UnixNano())
	return clock
}

func (f *fakeClock) now() time.Time          { return time.Unix(0, f.nanos.Load()).UTC() }
func (f *fakeClock) advance(d time.Duration) { f.nanos.Add(int64(d)) }

func newProtectedCache(t *testing.T, cfg Config) (*redisCache, *miniredis.Miniredis, *fakeClock) {
	t.Helper()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	c, ok := NewRedisFromClient(client, cfg).(*redisCache)
	require.True(t, ok)

	clock := newFakeClock()
	c.now = clock.now

	return c, server, clock
}

func TestNegativeCaching_AbsorbsRepeatedLookups(t *testing.T) {
	ctx := context.Background()
	c, _, _ := newProtectedCache(t, Config{NegativeTTL: time.Minute})

	var calls atomic.Int32
	load := func(context.Context) (any, error) {
		calls.Add(1)
		return nil, ErrNotFound
	}

	for i := range 5 {
		var got user
		err := c.GetOrLoad(ctx, "user:404", &got, time.Hour, load)
		assert.ErrorIs(t, err, ErrNotFound, "call %d", i)
	}

	assert.EqualValues(t, 1, calls.Load(), "only the first lookup should reach the origin")

	stats := c.Stats()
	assert.EqualValues(t, 1, stats.Misses)
	assert.EqualValues(t, 4, stats.NegativeHits, "the other four were absorbed by the tombstone")
}

func TestNegativeCaching_DisabledLetsEveryLookupThrough(t *testing.T) {
	ctx := context.Background()
	c, _, _ := newProtectedCache(t, Config{})

	var calls atomic.Int32
	load := func(context.Context) (any, error) {
		calls.Add(1)
		return nil, ErrNotFound
	}

	for range 3 {
		var got user
		assert.ErrorIs(t, c.GetOrLoad(ctx, "user:404", &got, time.Hour, load), ErrNotFound)
	}
	assert.EqualValues(t, 3, calls.Load())
}

func TestNegativeCaching_PerCallOverride(t *testing.T) {
	ctx := context.Background()
	c, _, _ := newProtectedCache(t, Config{NegativeTTL: time.Hour})

	var calls atomic.Int32
	load := func(context.Context) (any, error) {
		calls.Add(1)
		return nil, ErrNotFound
	}

	var got user
	assert.ErrorIs(t, c.GetOrLoad(ctx, "k", &got, time.Hour, load, WithNegativeTTL(0)), ErrNotFound)
	assert.ErrorIs(t, c.GetOrLoad(ctx, "k", &got, time.Hour, load, WithNegativeTTL(0)), ErrNotFound)
	assert.EqualValues(t, 2, calls.Load())

	exists, err := c.Exists(ctx, "k")
	require.NoError(t, err)
	assert.False(t, exists, "nothing should have been stored")
}

func TestNegativeCaching_TombstoneExpiresAndCanBeInvalidated(t *testing.T) {
	ctx := context.Background()
	c, server, _ := newProtectedCache(t, Config{NegativeTTL: 30 * time.Second})

	missing := func(context.Context) (any, error) { return nil, ErrNotFound }
	var got user
	require.ErrorIs(t, c.GetOrLoad(ctx, "k", &got, time.Hour, missing), ErrNotFound)

	t.Run("Get reports the tombstone distinctly from a miss", func(t *testing.T) {
		err := c.Get(ctx, "k", &got)
		assert.ErrorIs(t, err, ErrNotFound)
		assert.NotErrorIs(t, err, ErrCacheMiss)
	})

	t.Run("Delete clears it", func(t *testing.T) {
		require.NoError(t, c.Delete(ctx, "k"))
		assert.ErrorIs(t, c.Get(ctx, "k", &got), ErrCacheMiss)
	})

	t.Run("it expires by itself", func(t *testing.T) {
		require.ErrorIs(t, c.GetOrLoad(ctx, "k", &got, time.Hour, missing), ErrNotFound)
		server.FastForward(31 * time.Second)
		assert.ErrorIs(t, c.Get(ctx, "k", &got), ErrCacheMiss)
	})
}

func TestNegativeCaching_TombstoneGivesWayToARealValue(t *testing.T) {
	ctx := context.Background()
	c, _, _ := newProtectedCache(t, Config{NegativeTTL: time.Minute})

	var got user
	require.ErrorIs(t, c.GetOrLoad(ctx, "user:7", &got, time.Hour,
		func(context.Context) (any, error) { return nil, ErrNotFound }), ErrNotFound)

	require.NoError(t, c.Delete(ctx, "user:7"))

	require.NoError(t, c.GetOrLoad(ctx, "user:7", &got, time.Hour,
		func(context.Context) (any, error) { return user{ID: 7, Name: "kirk"}, nil }))
	assert.Equal(t, "kirk", got.Name)
}

func TestStaleWhileRevalidate_ServesStaleAndRefreshesBehind(t *testing.T) {
	ctx := context.Background()
	c, _, clock := newProtectedCache(t, Config{})

	version := atomic.Int32{}
	version.Store(1)
	refreshed := make(chan struct{}, 1)

	load := func(context.Context) (any, error) {
		v := version.Load()
		select {
		case refreshed <- struct{}{}:
		default:
		}
		return user{ID: int64(v), Name: fmt.Sprintf("v%d", v)}, nil
	}

	stale := WithStaleWhileRevalidate(5 * time.Minute)

	var got user
	require.NoError(t, c.GetOrLoad(ctx, "k", &got, time.Minute, load, stale))
	assert.Equal(t, "v1", got.Name)
	<-refreshed

	version.Store(2)
	clock.advance(2 * time.Minute)

	require.NoError(t, c.GetOrLoad(ctx, "k", &got, time.Minute, load, stale))
	assert.Equal(t, "v1", got.Name, "the caller must not wait for the rebuild")

	select {
	case <-refreshed:
	case <-time.After(2 * time.Second):
		t.Fatal("background refresh never ran")
	}

	require.Eventually(t, func() bool {
		var latest user
		return c.GetOrLoad(ctx, "k", &latest, time.Minute, load, stale) == nil && latest.Name == "v2"
	}, 2*time.Second, 20*time.Millisecond, "the refreshed value should land")

	stats := c.Stats()
	assert.EqualValues(t, 1, stats.StaleHits)
	assert.EqualValues(t, 1, stats.Refreshes)
}

func TestStaleWhileRevalidate_StopsAtTheGraceWindow(t *testing.T) {
	ctx := context.Background()
	c, server, clock := newProtectedCache(t, Config{})

	var calls atomic.Int32
	load := func(context.Context) (any, error) {
		calls.Add(1)
		return user{ID: 1, Name: "value"}, nil
	}

	stale := WithStaleWhileRevalidate(30 * time.Second)
	var got user
	require.NoError(t, c.GetOrLoad(ctx, "k", &got, time.Minute, load, stale))

	server.FastForward(2 * time.Minute)
	clock.advance(2 * time.Minute)

	before := calls.Load()
	require.NoError(t, c.GetOrLoad(ctx, "k", &got, time.Minute, load, stale))
	assert.Greater(t, calls.Load(), before, "a physically expired key must be rebuilt synchronously")
	assert.EqualValues(t, 2, c.Stats().Misses, "the cold read and the expired read both missed")
}

func TestStaleWhileRevalidate_PhysicalTTLCoversTheGrace(t *testing.T) {
	ctx := context.Background()
	c, _, _ := newProtectedCache(t, Config{})

	var got user
	require.NoError(t, c.GetOrLoad(ctx, "k", &got, time.Minute, func(context.Context) (any, error) {
		return user{ID: 1}, nil
	}, WithStaleWhileRevalidate(5*time.Minute)))

	ttl, err := c.TTL(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, 6*time.Minute, ttl, "physical expiry is ttl + grace")
}

func TestStaleWhileRevalidate_OneRefreshPerKey(t *testing.T) {
	ctx := context.Background()
	c, _, clock := newProtectedCache(t, Config{})

	var loads atomic.Int32
	release := make(chan struct{})
	load := func(context.Context) (any, error) {
		if loads.Add(1) > 1 {
			<-release
		}
		return user{ID: 1, Name: "value"}, nil
	}

	stale := WithStaleWhileRevalidate(5 * time.Minute)
	var got user
	require.NoError(t, c.GetOrLoad(ctx, "k", &got, time.Minute, load, stale))
	clock.advance(2 * time.Minute)

	var wg sync.WaitGroup
	for range 30 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var stale2 user
			_ = c.GetOrLoad(ctx, "k", &stale2, time.Minute, load, stale)
		}()
	}
	wg.Wait()

	assert.EqualValues(t, 1, c.Stats().Refreshes, "30 stale reads, one refresh")
	close(release)
}

func TestSharedFlight_OneReplicaLoadsForTheFleet(t *testing.T) {
	ctx := context.Background()
	server := miniredis.RunT(t)

	replica := func() *redisCache {
		client := redis.NewClient(&redis.Options{Addr: server.Addr()})
		t.Cleanup(func() { _ = client.Close() })
		c, ok := NewRedisFromClient(client, Config{}).(*redisCache)
		require.True(t, ok)
		return c
	}

	first, second := replica(), replica()

	var firstCalls, secondCalls atomic.Int32
	holding := make(chan struct{})
	release := make(chan struct{})

	shared := WithSharedFlight(5 * time.Second)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		var got user
		err := first.GetOrLoad(ctx, "k", &got, time.Minute, func(context.Context) (any, error) {
			firstCalls.Add(1)
			close(holding)
			<-release
			return user{ID: 1, Name: "loaded once"}, nil
		}, shared)
		assert.NoError(t, err)
	}()

	<-holding

	go func() {
		defer wg.Done()
		var got user
		err := second.GetOrLoad(ctx, "k", &got, time.Minute, func(context.Context) (any, error) {
			secondCalls.Add(1)
			return user{ID: 2, Name: "should not happen"}, nil
		}, shared)
		assert.NoError(t, err)
		assert.Equal(t, "loaded once", got.Name, "the waiter should get the holder's value")
	}()

	time.Sleep(100 * time.Millisecond)
	close(release)
	wg.Wait()

	assert.EqualValues(t, 1, firstCalls.Load())
	assert.EqualValues(t, 0, secondCalls.Load(), "the origin should be called once across replicas")
	assert.EqualValues(t, 1, second.Stats().SharedFlightWaits)
}

func TestSharedFlight_WaiterLoadsAnywayWhenTheHolderNeverFinishes(t *testing.T) {
	ctx := context.Background()
	c, _, _ := newProtectedCache(t, Config{})

	taken, err := c.SetNX(ctx, "lock:k", "someone-else", 200*time.Millisecond)
	require.NoError(t, err)
	require.True(t, taken)

	var calls atomic.Int32
	var got user
	err = c.GetOrLoad(ctx, "k", &got, time.Minute, func(context.Context) (any, error) {
		calls.Add(1)
		return user{ID: 1, Name: "loaded anyway"}, nil
	}, WithSharedFlight(200*time.Millisecond))

	require.NoError(t, err)
	assert.Equal(t, "loaded anyway", got.Name)
	assert.EqualValues(t, 1, calls.Load(), "a duplicated query beats a failed request")
}

func TestSharedFlight_ReleaseOnlyRemovesItsOwnLock(t *testing.T) {
	ctx := context.Background()
	c, _, _ := newProtectedCache(t, Config{})

	_, acquired, err := c.acquireLock(ctx, "k", time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)

	c.releaseLock("k", "not-the-current-token")

	_, stillHeld, err := c.acquireLock(ctx, "k", time.Minute)
	require.NoError(t, err)
	assert.False(t, stillHeld, "the lock must survive a release from a non-owner")
}

func TestStats_HitRate(t *testing.T) {
	tests := []struct {
		name  string
		stats Stats
		want  float64
	}{
		{name: "no reads", stats: Stats{}, want: 0},
		{name: "all hits", stats: Stats{Hits: 10}, want: 1},
		{name: "all misses", stats: Stats{Misses: 10}, want: 0},
		{name: "half", stats: Stats{Hits: 5, Misses: 5}, want: 0.5},
		{
			name:  "tombstone and stale reads count as hits",
			stats: Stats{Hits: 1, NegativeHits: 1, StaleHits: 2, Misses: 4},
			want:  0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.want, tt.stats.HitRate(), 1e-9)
		})
	}
}

func TestOnError_SurfacesBackgroundRefreshFailures(t *testing.T) {
	ctx := context.Background()

	reported := make(chan error, 1)
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	c, ok := NewRedisFromClient(client, Config{
		OnError: func(_ string, err error) {
			select {
			case reported <- err:
			default:
			}
		},
	}).(*redisCache)
	require.True(t, ok)

	clock := newFakeClock()
	c.now = clock.now

	origin := errors.New("origin is down")
	var attempts atomic.Int32
	load := func(context.Context) (any, error) {
		if attempts.Add(1) == 1 {
			return user{ID: 1, Name: "first"}, nil
		}
		return nil, origin
	}

	stale := WithStaleWhileRevalidate(5 * time.Minute)
	var got user
	require.NoError(t, c.GetOrLoad(ctx, "k", &got, time.Minute, load, stale))

	clock.advance(2 * time.Minute)
	require.NoError(t, c.GetOrLoad(ctx, "k", &got, time.Minute, load, stale))
	assert.Equal(t, "first", got.Name, "the stale value is still served")

	select {
	case err := <-reported:
		assert.ErrorIs(t, err, origin)
	case <-time.After(2 * time.Second):
		t.Fatal("the refresh failure was never reported")
	}
}

func TestGenericGetOrLoad(t *testing.T) {
	ctx := context.Background()
	c, _, _ := newProtectedCache(t, Config{NegativeTTL: time.Minute})

	t.Run("returns a typed value", func(t *testing.T) {
		got, err := GetOrLoad(ctx, c, "user:7", time.Hour, func(context.Context) (*user, error) {
			return &user{ID: 7, Name: "kirk"}, nil
		})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "kirk", got.Name)
	})

	t.Run("serves the second call from cache", func(t *testing.T) {
		got, err := GetOrLoad(ctx, c, "user:7", time.Hour, func(context.Context) (*user, error) {
			t.Fatal("loader must not run on a hit")
			return nil, nil
		})
		require.NoError(t, err)
		assert.Equal(t, "kirk", got.Name)
	})

	t.Run("ErrNotFound comes back with the zero value", func(t *testing.T) {
		got, err := GetOrLoad(ctx, c, "user:404", time.Hour, func(context.Context) (*user, error) {
			return nil, ErrNotFound
		})
		assert.ErrorIs(t, err, ErrNotFound)
		assert.Nil(t, got)
	})

	t.Run("nil loader is rejected", func(t *testing.T) {
		_, err := GetOrLoad[*user](ctx, c, "k", time.Hour, nil)
		assert.Error(t, err)
	})
}

func TestGenericGet(t *testing.T) {
	ctx := context.Background()
	c, _, _ := newProtectedCache(t, Config{})

	require.NoError(t, c.Set(ctx, "k", user{ID: 1, Name: "kirk"}, time.Minute))

	got, err := Get[user](ctx, c, "k")
	require.NoError(t, err)
	assert.Equal(t, "kirk", got.Name)

	_, err = Get[user](ctx, c, "absent")
	assert.ErrorIs(t, err, ErrCacheMiss)
}

func TestEnvelope_UnrecognizedPayloadReadsAsAMiss(t *testing.T) {
	ctx := context.Background()
	c, server, _ := newProtectedCache(t, Config{})

	require.NoError(t, server.Set("legacy", `{"id":1,"name":"written by an older build"}`))

	var got user
	assert.ErrorIs(t, c.Get(ctx, "legacy", &got), ErrCacheMiss)
}
