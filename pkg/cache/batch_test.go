package cache

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrLoadBatch(t *testing.T) {
	ctx := context.Background()

	t.Run("cold batch calls the loader once for every key", func(t *testing.T) {
		c, _, _ := newProtectedCache(t, Config{})
		var calls atomic.Int32

		got, err := GetOrLoadBatch(ctx, c, []string{"a", "b", "c"}, time.Hour,
			func(_ context.Context, missing []string) (map[string]user, error) {
				calls.Add(1)
				assert.ElementsMatch(t, []string{"a", "b", "c"}, missing)
				return map[string]user{
					"a": {ID: 1, Name: "a"},
					"b": {ID: 2, Name: "b"},
					"c": {ID: 3, Name: "c"},
				}, nil
			})

		require.NoError(t, err)
		assert.Len(t, got, 3)
		assert.Equal(t, "b", got["b"].Name)
		assert.EqualValues(t, 1, calls.Load(), "one origin call for the whole batch")
	})

	t.Run("a warm batch does not call the loader at all", func(t *testing.T) {
		c, _, _ := newProtectedCache(t, Config{})
		require.NoError(t, c.SetBatch(ctx, map[string]any{
			"a": user{ID: 1, Name: "a"},
			"b": user{ID: 2, Name: "b"},
		}, time.Hour))

		got, err := GetOrLoadBatch(ctx, c, []string{"a", "b"}, time.Hour,
			func(context.Context, []string) (map[string]user, error) {
				t.Fatal("loader must not run when everything is cached")
				return nil, nil
			})

		require.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("only the missing keys reach the loader", func(t *testing.T) {
		c, _, _ := newProtectedCache(t, Config{})
		require.NoError(t, c.SetBatch(ctx, map[string]any{"a": user{ID: 1, Name: "cached"}}, time.Hour))

		got, err := GetOrLoadBatch(ctx, c, []string{"a", "b"}, time.Hour,
			func(_ context.Context, missing []string) (map[string]user, error) {
				assert.Equal(t, []string{"b"}, missing, "the cached key must not be asked for again")
				return map[string]user{"b": {ID: 2, Name: "loaded"}}, nil
			})

		require.NoError(t, err)
		assert.Equal(t, "cached", got["a"].Name)
		assert.Equal(t, "loaded", got["b"].Name)
	})

	t.Run("omitted keys are tombstoned", func(t *testing.T) {
		c, _, _ := newProtectedCache(t, Config{NegativeTTL: time.Minute})
		var calls atomic.Int32

		load := func(_ context.Context, missing []string) (map[string]user, error) {
			calls.Add(1)
			result := map[string]user{}
			for _, key := range missing {
				if key == "exists" {
					result[key] = user{ID: 1, Name: "here"}
				}
			}
			return result, nil
		}

		for range 3 {
			got, err := GetOrLoadBatch(ctx, c, []string{"exists", "absent"}, time.Hour, load)
			require.NoError(t, err)
			assert.Len(t, got, 1)
			assert.NotContains(t, got, "absent")
		}

		assert.EqualValues(t, 1, calls.Load(), "the tombstone should stop the repeat lookups")
	})

	t.Run("without a negative TTL the absent key is asked for every time", func(t *testing.T) {
		c, _, _ := newProtectedCache(t, Config{})
		var calls atomic.Int32

		load := func(context.Context, []string) (map[string]user, error) {
			calls.Add(1)
			return map[string]user{}, nil
		}
		for range 3 {
			_, err := GetOrLoadBatch(ctx, c, []string{"absent"}, time.Hour, load)
			require.NoError(t, err)
		}
		assert.EqualValues(t, 3, calls.Load())
	})

	t.Run("duplicate keys collapse", func(t *testing.T) {
		c, _, _ := newProtectedCache(t, Config{})

		got, err := GetOrLoadBatch(ctx, c, []string{"a", "a", "a"}, time.Hour,
			func(_ context.Context, missing []string) (map[string]user, error) {
				assert.Equal(t, []string{"a"}, missing)
				return map[string]user{"a": {ID: 1}}, nil
			})

		require.NoError(t, err)
		assert.Len(t, got, 1)
	})

	t.Run("loader error propagates and nothing is cached", func(t *testing.T) {
		c, _, _ := newProtectedCache(t, Config{})
		sentinel := errors.New("origin unavailable")

		_, err := GetOrLoadBatch(ctx, c, []string{"a"}, time.Hour,
			func(context.Context, []string) (map[string]user, error) { return nil, sentinel })
		assert.ErrorIs(t, err, sentinel)

		exists, existsErr := c.Exists(ctx, "a")
		require.NoError(t, existsErr)
		assert.False(t, exists)
	})

	t.Run("empty input short-circuits", func(t *testing.T) {
		c, _, _ := newProtectedCache(t, Config{})

		got, err := GetOrLoadBatch(ctx, c, nil, time.Hour,
			func(context.Context, []string) (map[string]user, error) {
				t.Fatal("loader must not run for an empty batch")
				return nil, nil
			})
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("nil loader is rejected", func(t *testing.T) {
		c, _, _ := newProtectedCache(t, Config{})
		_, err := GetOrLoadBatch[user](ctx, c, []string{"a"}, time.Hour, nil)
		assert.Error(t, err)
	})

	t.Run("degrades to the origin when the cache is down", func(t *testing.T) {
		c, server, _ := newProtectedCache(t, Config{})
		server.Close()

		got, err := GetOrLoadBatch(ctx, c, []string{"a"}, time.Hour,
			func(context.Context, []string) (map[string]user, error) {
				return map[string]user{"a": {ID: 1, Name: "origin"}}, nil
			})

		require.NoError(t, err)
		assert.Equal(t, "origin", got["a"].Name)
	})
}

func TestGetBatch(t *testing.T) {
	ctx := context.Background()
	c, _, _ := newProtectedCache(t, Config{NegativeTTL: time.Minute})

	require.NoError(t, c.SetBatch(ctx, map[string]any{
		"a": user{ID: 1, Name: "a"},
		"b": user{ID: 2, Name: "b"},
	}, time.Hour))
	require.NoError(t, c.SetMissingBatch(ctx, []string{"gone"}))

	t.Run("raw entries distinguish the three states", func(t *testing.T) {
		entries, err := c.GetBatch(ctx, []string{"a", "gone", "never-cached"})
		require.NoError(t, err)

		require.Contains(t, entries, "a")
		assert.False(t, entries["a"].Missing)
		assert.NotEmpty(t, entries["a"].Payload)

		require.Contains(t, entries, "gone")
		assert.True(t, entries["gone"].Missing, "a tombstone is present but empty")

		assert.NotContains(t, entries, "never-cached", "an uncached key is simply absent")
	})

	t.Run("the typed form skips tombstones and misses", func(t *testing.T) {
		got, err := GetBatch[user](ctx, c, []string{"a", "b", "gone", "never-cached"})
		require.NoError(t, err)

		assert.Len(t, got, 2)
		assert.Equal(t, "a", got["a"].Name)
		assert.NotContains(t, got, "gone")
	})

	t.Run("empty input", func(t *testing.T) {
		entries, err := c.GetBatch(ctx, nil)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

func TestSetBatch_JittersEachKeySeparately(t *testing.T) {
	ctx := context.Background()
	c, _, _ := newProtectedCache(t, Config{TTLJitter: 0.2})

	values := map[string]any{}
	for i := range 30 {
		values[string(rune('a'+i%26))+string(rune('a'+i/26))] = user{ID: int64(i)}
	}
	require.NoError(t, c.SetBatch(ctx, values, 100*time.Second))

	spread := map[time.Duration]struct{}{}
	for key := range values {
		ttl, err := c.TTL(ctx, key)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, ttl, 80*time.Second)
		assert.LessOrEqual(t, ttl, 120*time.Second)
		spread[ttl] = struct{}{}
	}
	assert.Greater(t, len(spread), 1, "the batch must not land on a single expiry")
}

func TestSetMissingBatch_RespectsTheNegativeTTLSwitch(t *testing.T) {
	ctx := context.Background()

	t.Run("disabled writes nothing", func(t *testing.T) {
		c, _, _ := newProtectedCache(t, Config{})
		require.NoError(t, c.SetMissingBatch(ctx, []string{"a"}))

		exists, err := c.Exists(ctx, "a")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("per-call override wins", func(t *testing.T) {
		c, _, _ := newProtectedCache(t, Config{})
		require.NoError(t, c.SetMissingBatch(ctx, []string{"a"}, WithNegativeTTL(time.Minute)))

		var got user
		assert.ErrorIs(t, c.Get(ctx, "a", &got), ErrNotFound)
	})
}

func TestDedupe(t *testing.T) {
	assert.Nil(t, dedupe(nil))
	assert.Equal(t, []string{"a"}, dedupe([]string{"a"}))
	assert.Equal(t, []string{"a", "b"}, dedupe([]string{"a", "b", "a", "b"}),
		"order is preserved so the loader sees a stable request")
}

func TestNoopCache_Batch(t *testing.T) {
	ctx := context.Background()
	c := NewNoop()

	entries, err := c.GetBatch(ctx, []string{"a", "b"})
	require.NoError(t, err)
	assert.Empty(t, entries)

	assert.NoError(t, c.SetBatch(ctx, map[string]any{"a": 1}, time.Minute))
	assert.NoError(t, c.SetMissingBatch(ctx, []string{"a"}))

	var calls int
	got, err := GetOrLoadBatch(ctx, c, []string{"a"}, time.Minute,
		func(context.Context, []string) (map[string]user, error) {
			calls++
			return map[string]user{"a": {ID: 1}}, nil
		})
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, 1, calls)
}
