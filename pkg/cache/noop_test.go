package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoopCache_ReadsAlwaysMiss(t *testing.T) {
	ctx := context.Background()
	c := NewNoop()

	require.NoError(t, c.Set(ctx, "k", user{ID: 1}, time.Minute), "writes are accepted")

	var got user
	assert.ErrorIs(t, c.Get(ctx, "k", &got), ErrCacheMiss, "but nothing is stored")

	exists, err := c.Exists(ctx, "k")
	require.NoError(t, err)
	assert.False(t, exists)

	_, err = c.TTL(ctx, "k")
	assert.ErrorIs(t, err, ErrCacheMiss)
}

func TestNoopCache_GetOrLoadAlwaysCallsLoader(t *testing.T) {
	ctx := context.Background()
	c := NewNoop()

	calls := 0
	load := func(context.Context) (any, error) {
		calls++
		return user{ID: 3, Name: "origin"}, nil
	}

	for range 3 {
		var got user
		require.NoError(t, c.GetOrLoad(ctx, "k", &got, time.Minute, load))
		assert.Equal(t, user{ID: 3, Name: "origin"}, got)
	}
	assert.Equal(t, 3, calls, "no caching means no reuse")
}

func TestNoopCache_GetOrLoadPropagatesLoaderError(t *testing.T) {
	ctx := context.Background()
	sentinel := errors.New("origin unavailable")

	var got user
	err := NewNoop().GetOrLoad(ctx, "k", &got, time.Minute, func(context.Context) (any, error) {
		return nil, sentinel
	})
	assert.ErrorIs(t, err, sentinel)
}

func TestNoopCache_ValidatesDest(t *testing.T) {
	ctx := context.Background()
	c := NewNoop()

	assert.Error(t, c.Get(ctx, "k", nil))
	assert.Error(t, c.GetOrLoad(ctx, "k", nil, time.Minute, func(context.Context) (any, error) {
		return nil, nil
	}))
	var got user
	assert.Error(t, c.GetOrLoad(ctx, "k", &got, time.Minute, nil))
}

func TestNoopCache_ProvidesNoSharedState(t *testing.T) {
	ctx := context.Background()
	c := NewNoop()

	t.Run("SetNX never blocks a second writer", func(t *testing.T) {
		first, err := c.SetNX(ctx, "lock", "a", time.Minute)
		require.NoError(t, err)
		second, err := c.SetNX(ctx, "lock", "b", time.Minute)
		require.NoError(t, err)

		assert.True(t, first)
		assert.True(t, second, "no mutual exclusion without shared state")
	})

	t.Run("Increment never accumulates", func(t *testing.T) {
		for range 3 {
			got, err := c.Increment(ctx, "hits", 1)
			require.NoError(t, err)
			assert.EqualValues(t, 1, got, "each call reports only its own delta")
		}
	})
}

func TestNoopCache_LifecycleIsInert(t *testing.T) {
	ctx := context.Background()
	c := NewNoop()

	assert.NoError(t, c.Ping(ctx))
	assert.NoError(t, c.Delete(ctx, "a", "b"))

	ok, err := c.Expire(ctx, "k", time.Minute)
	require.NoError(t, err)
	assert.False(t, ok)

	assert.NoError(t, c.Close())
}
