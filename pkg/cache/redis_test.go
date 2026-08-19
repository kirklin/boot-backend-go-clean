package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type user struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func newTestCache(t *testing.T, cfg Config) (Cache, *miniredis.Miniredis) {
	t.Helper()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return NewRedisFromClient(client, cfg), server
}

func TestRedisCache_SetGet(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestCache(t, Config{})

	want := user{ID: 7, Name: "kirk"}
	require.NoError(t, c.Set(ctx, "user:7", want, time.Minute))

	var got user
	require.NoError(t, c.Get(ctx, "user:7", &got))
	assert.Equal(t, want, got)
}

func TestRedisCache_GetMiss(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestCache(t, Config{})

	var got user
	err := c.Get(ctx, "absent", &got)
	assert.ErrorIs(t, err, ErrCacheMiss)
}

func TestRedisCache_GetRejectsBadDest(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestCache(t, Config{})
	require.NoError(t, c.Set(ctx, "k", "v", time.Minute))

	t.Run("nil", func(t *testing.T) {
		assert.Error(t, c.Get(ctx, "k", nil))
	})
	t.Run("non-pointer", func(t *testing.T) {
		var got user
		assert.Error(t, c.Get(ctx, "k", got))
	})
	t.Run("typed nil pointer", func(t *testing.T) {
		var got *user
		assert.Error(t, c.Get(ctx, "k", got))
	})
}

func TestRedisCache_SetZeroTTLNeverExpires(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestCache(t, Config{})

	require.NoError(t, c.Set(ctx, "forever", "v", 0))

	ttl, err := c.TTL(ctx, "forever")
	require.NoError(t, err)
	assert.Equal(t, NoExpiry, ttl)
}

func TestRedisCache_SetNX(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestCache(t, Config{})

	ok, err := c.SetNX(ctx, "lock", "owner-a", time.Minute)
	require.NoError(t, err)
	assert.True(t, ok, "first writer should acquire")

	ok, err = c.SetNX(ctx, "lock", "owner-b", time.Minute)
	require.NoError(t, err)
	assert.False(t, ok, "second writer should be refused")

	var owner string
	require.NoError(t, c.Get(ctx, "lock", &owner))
	assert.Equal(t, "owner-a", owner, "refused write must not overwrite")
}

func TestRedisCache_SetNXIgnoresJitter(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestCache(t, Config{TTLJitter: 0.5})

	_, err := c.SetNX(ctx, "cooldown", "1", time.Minute)
	require.NoError(t, err)

	ttl, err := c.TTL(ctx, "cooldown")
	require.NoError(t, err)
	assert.Equal(t, time.Minute, ttl)
}

func TestRedisCache_Delete(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestCache(t, Config{})

	require.NoError(t, c.Set(ctx, "a", 1, time.Minute))
	require.NoError(t, c.Set(ctx, "b", 2, time.Minute))

	require.NoError(t, c.Delete(ctx, "a", "b", "never-existed"))

	for _, key := range []string{"a", "b"} {
		exists, err := c.Exists(ctx, key)
		require.NoError(t, err)
		assert.False(t, exists, "%s should be gone", key)
	}

	assert.NoError(t, c.Delete(ctx), "deleting nothing is a no-op")
}

func TestRedisCache_Increment(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestCache(t, Config{})

	got, err := c.Increment(ctx, "hits", 1)
	require.NoError(t, err)
	assert.EqualValues(t, 1, got, "missing key counts from zero")

	got, err = c.Increment(ctx, "hits", 4)
	require.NoError(t, err)
	assert.EqualValues(t, 5, got)

	got, err = c.Increment(ctx, "hits", -2)
	require.NoError(t, err)
	assert.EqualValues(t, 3, got)
}

func TestRedisCache_Expire(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestCache(t, Config{})

	ok, err := c.Expire(ctx, "absent", time.Minute)
	require.NoError(t, err)
	assert.False(t, ok, "cannot expire a key that is not there")

	require.NoError(t, c.Set(ctx, "present", "v", 0))
	ok, err = c.Expire(ctx, "present", time.Minute)
	require.NoError(t, err)
	assert.True(t, ok)

	ttl, err := c.TTL(ctx, "present")
	require.NoError(t, err)
	assert.Equal(t, time.Minute, ttl)
}

func TestRedisCache_TTL(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestCache(t, Config{})

	t.Run("missing key", func(t *testing.T) {
		_, err := c.TTL(ctx, "absent")
		assert.ErrorIs(t, err, ErrCacheMiss)
	})

	t.Run("no expiry", func(t *testing.T) {
		require.NoError(t, c.Set(ctx, "forever", "v", 0))
		ttl, err := c.TTL(ctx, "forever")
		require.NoError(t, err)
		assert.Equal(t, NoExpiry, ttl)
	})

	t.Run("with expiry", func(t *testing.T) {
		require.NoError(t, c.Set(ctx, "temporary", "v", 90*time.Second))
		ttl, err := c.TTL(ctx, "temporary")
		require.NoError(t, err)
		assert.Equal(t, 90*time.Second, ttl)
	})
}

func TestRedisCache_KeyPrefix(t *testing.T) {
	ctx := context.Background()
	c, server := newTestCache(t, Config{KeyPrefix: "boot"})

	require.NoError(t, c.Set(ctx, "user:7", "v", time.Minute))

	assert.True(t, server.Exists("boot:user:7"), "prefix should be applied on write")
	assert.False(t, server.Exists("user:7"), "unprefixed key should not be written")

	var got string
	require.NoError(t, c.Get(ctx, "user:7", &got), "reads take the bare key")
	assert.Equal(t, "v", got)
}

func TestRedisCache_JitterStaysWithinBounds(t *testing.T) {
	ctx := context.Background()

	var (
		base     = 100 * time.Second
		fraction = 0.1
	)
	c, _ := newTestCache(t, Config{TTLJitter: fraction})

	lower := time.Duration(float64(base) * (1 - fraction))
	upper := time.Duration(float64(base) * (1 + fraction))

	spread := make(map[time.Duration]struct{})
	for i := range 50 {
		key := "k" + string(rune('a'+i%26)) + string(rune('a'+i/26))
		require.NoError(t, c.Set(ctx, key, "v", base))

		ttl, err := c.TTL(ctx, key)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, ttl, lower)
		assert.LessOrEqual(t, ttl, upper)
		spread[ttl] = struct{}{}
	}

	assert.Greater(t, len(spread), 1, "jitter should produce more than one expiry")
}

func TestRedisCache_NoJitterIsExact(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestCache(t, Config{})

	require.NoError(t, c.Set(ctx, "k", "v", time.Minute))

	ttl, err := c.TTL(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, time.Minute, ttl)
}

func TestRedisCache_GetOrLoad(t *testing.T) {
	ctx := context.Background()

	t.Run("miss calls loader and caches the result", func(t *testing.T) {
		c, _ := newTestCache(t, Config{})
		var calls atomic.Int32

		load := func(context.Context) (any, error) {
			calls.Add(1)
			return user{ID: 1, Name: "loaded"}, nil
		}

		var got user
		require.NoError(t, c.GetOrLoad(ctx, "user:1", &got, time.Minute, load))
		assert.Equal(t, user{ID: 1, Name: "loaded"}, got)
		assert.EqualValues(t, 1, calls.Load())

		var cached user
		require.NoError(t, c.Get(ctx, "user:1", &cached), "result should be written back")
		assert.Equal(t, got, cached)
	})

	t.Run("hit skips the loader", func(t *testing.T) {
		c, _ := newTestCache(t, Config{})
		require.NoError(t, c.Set(ctx, "user:1", user{ID: 1, Name: "cached"}, time.Minute))

		var got user
		err := c.GetOrLoad(ctx, "user:1", &got, time.Minute, func(context.Context) (any, error) {
			t.Fatal("loader must not run on a hit")
			return nil, nil
		})
		require.NoError(t, err)
		assert.Equal(t, "cached", got.Name)
	})

	t.Run("loader error propagates and nothing is cached", func(t *testing.T) {
		c, _ := newTestCache(t, Config{})
		sentinel := errors.New("origin unavailable")

		var got user
		err := c.GetOrLoad(ctx, "user:1", &got, time.Minute, func(context.Context) (any, error) {
			return nil, sentinel
		})
		assert.ErrorIs(t, err, sentinel)

		exists, existsErr := c.Exists(ctx, "user:1")
		require.NoError(t, existsErr)
		assert.False(t, exists)
	})

	t.Run("nil loader is rejected", func(t *testing.T) {
		c, _ := newTestCache(t, Config{})
		var got user
		assert.Error(t, c.GetOrLoad(ctx, "k", &got, time.Minute, nil))
	})
}

func TestRedisCache_GetOrLoadCollapsesConcurrentMisses(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestCache(t, Config{})

	const callers = 20
	var calls atomic.Int32
	release := make(chan struct{})

	load := func(context.Context) (any, error) {
		calls.Add(1)
		<-release
		return user{ID: 1, Name: "loaded"}, nil
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make([]user, callers)
	errs := make([]error, callers)

	for i := range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs[i] = c.GetOrLoad(ctx, "user:1", &results[i], time.Minute, load)
		}()
	}

	close(start)
	time.Sleep(100 * time.Millisecond)
	close(release)
	wg.Wait()

	assert.EqualValues(t, 1, calls.Load(), "singleflight should collapse the burst")
	for i := range callers {
		require.NoError(t, errs[i])
		assert.Equal(t, user{ID: 1, Name: "loaded"}, results[i], "caller %d", i)
	}
}

func TestRedisCache_GetOrLoadDegradesWhenRedisIsDown(t *testing.T) {
	ctx := context.Background()
	c, server := newTestCache(t, Config{})
	server.Close()

	var calls atomic.Int32
	var got user
	err := c.GetOrLoad(ctx, "user:1", &got, time.Minute, func(context.Context) (any, error) {
		calls.Add(1)
		return user{ID: 1, Name: "origin"}, nil
	})

	require.NoError(t, err)
	assert.Equal(t, "origin", got.Name)
	assert.EqualValues(t, 1, calls.Load())
}

func TestRedisCache_Ping(t *testing.T) {
	ctx := context.Background()
	c, server := newTestCache(t, Config{})

	require.NoError(t, c.Ping(ctx))

	server.Close()
	assert.Error(t, c.Ping(ctx), "ping must fail once the server is gone")
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{name: "minimal", cfg: Config{Addr: "localhost:6379"}},
		{name: "jitter in range", cfg: Config{Addr: "localhost:6379", TTLJitter: 0.5}},
		{name: "missing addr", cfg: Config{}, wantErr: true},
		{name: "negative jitter", cfg: Config{Addr: "x:1", TTLJitter: -0.1}, wantErr: true},
		{name: "jitter of one", cfg: Config{Addr: "x:1", TTLJitter: 1}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestNormalizePrefix(t *testing.T) {
	assert.Equal(t, "", normalizePrefix(""))
	assert.Equal(t, "boot:", normalizePrefix("boot"))
	assert.Equal(t, "boot:", normalizePrefix("boot:"))
}

func TestNewRedis(t *testing.T) {
	t.Run("connects and works end to end", func(t *testing.T) {
		server := miniredis.RunT(t)

		c, err := NewRedis(Config{Addr: server.Addr(), KeyPrefix: "boot"})
		require.NoError(t, err)
		t.Cleanup(func() { _ = c.Close() })

		ctx := context.Background()
		require.NoError(t, c.Set(ctx, "k", user{ID: 1, Name: "kirk"}, time.Minute))

		var got user
		require.NoError(t, c.Get(ctx, "k", &got))
		assert.Equal(t, "kirk", got.Name)
		assert.True(t, server.Exists("boot:k"), "config should reach the implementation")
	})

	t.Run("unreachable server fails fast", func(t *testing.T) {
		_, err := NewRedis(Config{
			Addr:        "127.0.0.1:1",
			DialTimeout: 200 * time.Millisecond,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "127.0.0.1:1", "the message should name the address tried")
	})

	t.Run("rejects invalid config before dialing", func(t *testing.T) {
		_, err := NewRedis(Config{})
		assert.Error(t, err)

		_, err = NewRedis(Config{Addr: "127.0.0.1:6379", TTLJitter: 1.5})
		assert.Error(t, err)
	})

	t.Run("wrong password is rejected", func(t *testing.T) {
		server := miniredis.RunT(t)
		server.RequireAuth("correct-horse")

		_, err := NewRedis(Config{
			Addr:        server.Addr(),
			Password:    "wrong",
			DialTimeout: time.Second,
		})
		assert.Error(t, err)
	})
}
