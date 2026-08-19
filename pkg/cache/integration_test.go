package cache

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func integrationCache(t *testing.T) Cache {
	t.Helper()

	addr := os.Getenv("CACHE_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set CACHE_TEST_REDIS_ADDR to run cache integration tests")
	}

	c, err := NewRedis(Config{
		Addr:      addr,
		Password:  os.Getenv("CACHE_TEST_REDIS_PASSWORD"),
		KeyPrefix: "boot-integration",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	return c
}

func TestIntegration_RoundTrip(t *testing.T) {
	ctx := context.Background()
	c := integrationCache(t)
	key := "round-trip"
	t.Cleanup(func() { _ = c.Delete(ctx, key) })

	want := user{ID: 42, Name: "kirk"}
	require.NoError(t, c.Set(ctx, key, want, time.Minute))

	var got user
	require.NoError(t, c.Get(ctx, key, &got))
	assert.Equal(t, want, got)

	ttl, err := c.TTL(ctx, key)
	require.NoError(t, err)
	assert.Positive(t, ttl)
	assert.LessOrEqual(t, ttl, time.Minute)

	require.NoError(t, c.Delete(ctx, key))
	assert.ErrorIs(t, c.Get(ctx, key, &got), ErrCacheMiss)
}

func TestIntegration_SetNX(t *testing.T) {
	ctx := context.Background()
	c := integrationCache(t)
	key := "lock"
	t.Cleanup(func() { _ = c.Delete(ctx, key) })

	acquired, err := c.SetNX(ctx, key, "owner-a", time.Minute)
	require.NoError(t, err)
	assert.True(t, acquired)

	acquired, err = c.SetNX(ctx, key, "owner-b", time.Minute)
	require.NoError(t, err)
	assert.False(t, acquired)

	var owner string
	require.NoError(t, c.Get(ctx, key, &owner))
	assert.Equal(t, "owner-a", owner)

	ttl, err := c.TTL(ctx, key)
	require.NoError(t, err)
	assert.Positive(t, ttl, "the key must not be left without an expiry")
}

func TestIntegration_TTLSentinels(t *testing.T) {
	ctx := context.Background()
	c := integrationCache(t)

	t.Run("missing key", func(t *testing.T) {
		_, err := c.TTL(ctx, "definitely-not-here")
		assert.ErrorIs(t, err, ErrCacheMiss)
	})

	t.Run("no expiry", func(t *testing.T) {
		key := "persistent"
		t.Cleanup(func() { _ = c.Delete(ctx, key) })
		require.NoError(t, c.Set(ctx, key, "v", 0))

		ttl, err := c.TTL(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, NoExpiry, ttl)
	})
}

func TestIntegration_GetOrLoadCollapsesConcurrentMisses(t *testing.T) {
	ctx := context.Background()
	c := integrationCache(t)
	key := "cold"
	require.NoError(t, c.Delete(ctx, key))
	t.Cleanup(func() { _ = c.Delete(ctx, key) })

	const callers = 20
	var calls atomic.Int32
	release := make(chan struct{})

	load := func(context.Context) (any, error) {
		calls.Add(1)
		<-release
		return user{ID: 42, Name: "origin"}, nil
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			var got user
			if err := c.GetOrLoad(ctx, key, &got, time.Minute, load); err != nil {
				t.Error(err)
			}
		}()
	}

	close(start)
	time.Sleep(100 * time.Millisecond)
	close(release)
	wg.Wait()

	assert.EqualValues(t, 1, calls.Load())

	var cached user
	require.NoError(t, c.Get(ctx, key, &cached), "the flight should have written back")
	assert.Equal(t, "origin", cached.Name)
}

func TestIntegration_Counters(t *testing.T) {
	ctx := context.Background()
	c := integrationCache(t)
	key := "quota"
	require.NoError(t, c.Delete(ctx, key))
	t.Cleanup(func() { _ = c.Delete(ctx, key) })

	for i := 1; i <= 3; i++ {
		got, err := c.Increment(ctx, key, 1)
		require.NoError(t, err)
		assert.EqualValues(t, i, got)
	}

	set, err := c.Expire(ctx, key, time.Minute)
	require.NoError(t, err)
	assert.True(t, set)
}

func integrationCacheWith(t *testing.T, cfg Config) Cache {
	t.Helper()

	addr := os.Getenv("CACHE_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set CACHE_TEST_REDIS_ADDR to run cache integration tests")
	}

	cfg.Addr = addr
	cfg.Password = os.Getenv("CACHE_TEST_REDIS_PASSWORD")
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "boot-integration"
	}

	c, err := NewRedis(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	return c
}

func TestIntegration_NegativeCaching(t *testing.T) {
	ctx := context.Background()
	c := integrationCacheWith(t, Config{NegativeTTL: 30 * time.Second})

	key := "missing-row"
	require.NoError(t, c.Delete(ctx, key))
	t.Cleanup(func() { _ = c.Delete(ctx, key) })

	var calls atomic.Int32
	load := func(context.Context) (any, error) {
		calls.Add(1)
		return nil, ErrNotFound
	}

	for range 4 {
		var got user
		assert.ErrorIs(t, c.GetOrLoad(ctx, key, &got, time.Hour, load), ErrNotFound)
	}
	assert.EqualValues(t, 1, calls.Load(), "the tombstone should absorb the rest")

	present, err := c.Exists(ctx, key)
	require.NoError(t, err)
	assert.True(t, present)

	var got user
	assert.ErrorIs(t, c.Get(ctx, key, &got), ErrNotFound)

	ttl, err := c.TTL(ctx, key)
	require.NoError(t, err)
	assert.Positive(t, ttl)
	assert.LessOrEqual(t, ttl, 30*time.Second)
}

func TestIntegration_LockReleaseIsOwnerChecked(t *testing.T) {
	ctx := context.Background()
	c, ok := integrationCacheWith(t, Config{}).(*redisCache)
	require.True(t, ok)

	key := "lock-owner"
	t.Cleanup(func() { c.releaseLock(key, "cleanup") })

	token, acquired, err := c.acquireLock(ctx, key, time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)

	c.releaseLock(key, "some-other-token")

	_, stillHeld, err := c.acquireLock(ctx, key, time.Minute)
	require.NoError(t, err)
	assert.False(t, stillHeld, "a non-owner must not be able to release the lock")

	c.releaseLock(key, token)
	_, reacquired, err := c.acquireLock(ctx, key, time.Minute)
	require.NoError(t, err)
	assert.True(t, reacquired)
}

func TestIntegration_SharedFlightAcrossClients(t *testing.T) {
	ctx := context.Background()
	first := integrationCacheWith(t, Config{})
	second := integrationCacheWith(t, Config{})

	key := "shared-flight"
	require.NoError(t, first.Delete(ctx, key))
	t.Cleanup(func() { _ = first.Delete(ctx, key) })

	var firstCalls, secondCalls atomic.Int32
	holding := make(chan struct{})
	release := make(chan struct{})
	shared := WithSharedFlight(5 * time.Second)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		var got user
		err := first.GetOrLoad(ctx, key, &got, time.Minute, func(context.Context) (any, error) {
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
		err := second.GetOrLoad(ctx, key, &got, time.Minute, func(context.Context) (any, error) {
			secondCalls.Add(1)
			return user{ID: 2, Name: "should not happen"}, nil
		}, shared)
		assert.NoError(t, err)
		assert.Equal(t, "loaded once", got.Name)
	}()

	time.Sleep(150 * time.Millisecond)
	close(release)
	wg.Wait()

	assert.EqualValues(t, 1, firstCalls.Load())
	assert.EqualValues(t, 0, secondCalls.Load(), "one origin call for the whole fleet")
}

func TestIntegration_StaleWhileRevalidate(t *testing.T) {
	ctx := context.Background()
	c := integrationCacheWith(t, Config{})

	key := "stale-while-revalidate"
	require.NoError(t, c.Delete(ctx, key))
	t.Cleanup(func() { _ = c.Delete(ctx, key) })

	version := atomic.Int32{}
	version.Store(1)
	load := func(context.Context) (any, error) {
		v := version.Load()
		return user{ID: int64(v), Name: fmt.Sprintf("v%d", v)}, nil
	}

	stale := WithStaleWhileRevalidate(time.Hour)

	var got user
	require.NoError(t, c.GetOrLoad(ctx, key, &got, time.Second, load, stale))
	assert.Equal(t, "v1", got.Name)

	ttl, err := c.TTL(ctx, key)
	require.NoError(t, err)
	assert.Greater(t, ttl, 30*time.Minute)

	version.Store(2)
	time.Sleep(1200 * time.Millisecond)

	require.NoError(t, c.GetOrLoad(ctx, key, &got, time.Second, load, stale))
	assert.Equal(t, "v1", got.Name, "the stale value is served while the rebuild runs")

	require.Eventually(t, func() bool {
		var latest user
		return c.Get(ctx, key, &latest) == nil && latest.Name == "v2"
	}, 3*time.Second, 50*time.Millisecond, "the background refresh should land")
}

func TestIntegration_Batch(t *testing.T) {
	ctx := context.Background()
	c := integrationCacheWith(t, Config{NegativeTTL: 30 * time.Second})

	keys := []string{"batch:1", "batch:2", "batch:3"}
	require.NoError(t, c.Delete(ctx, keys...))
	t.Cleanup(func() { _ = c.Delete(ctx, keys...) })

	var calls atomic.Int32
	load := func(_ context.Context, missing []string) (map[string]user, error) {
		calls.Add(1)
		loaded := map[string]user{}
		for _, key := range missing {
			if key == "batch:3" {
				continue
			}
			loaded[key] = user{Name: key}
		}
		return loaded, nil
	}

	first, err := GetOrLoadBatch(ctx, c, keys, time.Hour, load)
	require.NoError(t, err)
	assert.Len(t, first, 2)
	assert.NotContains(t, first, "batch:3")

	second, err := GetOrLoadBatch(ctx, c, keys, time.Hour, load)
	require.NoError(t, err)
	assert.Len(t, second, 2)
	assert.EqualValues(t, 1, calls.Load(), "the second batch should not reach the origin")

	entries, err := c.GetBatch(ctx, keys)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
	assert.True(t, entries["batch:3"].Missing)

	for _, key := range []string{"batch:1", "batch:2"} {
		ttl, ttlErr := c.TTL(ctx, key)
		require.NoError(t, ttlErr)
		assert.Positive(t, ttl)
	}
}
