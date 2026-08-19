package cache

import (
	"context"
	"errors"
	"time"
)

type noopCache struct {
	stats counters
}

// NewNoop returns a Cache that stores nothing. Development and tests only
func NewNoop() Cache { return &noopCache{} }

func (c *noopCache) Get(_ context.Context, _ string, dest any) error {
	if err := validateDest(dest); err != nil {
		return err
	}
	c.stats.misses.Add(1)
	return ErrCacheMiss
}

func (c *noopCache) Set(_ context.Context, _ string, _ any, _ time.Duration) error {
	return nil
}

func (c *noopCache) SetNX(_ context.Context, _ string, _ any, _ time.Duration) (bool, error) {
	return true, nil
}

func (c *noopCache) GetOrLoad(
	ctx context.Context,
	_ string,
	dest any,
	_ time.Duration,
	load Loader,
	_ ...LoadOption,
) error {
	if err := validateDest(dest); err != nil {
		return err
	}
	if load == nil {
		return errors.New("cache: load must not be nil")
	}

	c.stats.misses.Add(1)
	c.stats.loads.Add(1)

	value, err := load(ctx)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			c.stats.loadErrors.Add(1)
		}
		return err
	}
	return transfer(value, dest)
}

func (c *noopCache) GetBatch(_ context.Context, keys []string) (map[string]Entry, error) {
	c.stats.misses.Add(uint64(len(keys)))
	return map[string]Entry{}, nil
}

func (c *noopCache) SetBatch(_ context.Context, _ map[string]any, _ time.Duration, _ ...LoadOption) error {
	return nil
}

func (c *noopCache) SetMissingBatch(_ context.Context, _ []string, _ ...LoadOption) error {
	return nil
}

func (c *noopCache) Delete(_ context.Context, _ ...string) error {
	return nil
}

func (c *noopCache) Exists(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (c *noopCache) Increment(_ context.Context, _ string, delta int64) (int64, error) {
	return delta, nil
}

func (c *noopCache) Expire(_ context.Context, _ string, _ time.Duration) (bool, error) {
	return false, nil
}

func (c *noopCache) TTL(_ context.Context, _ string) (time.Duration, error) {
	return 0, ErrCacheMiss
}

func (c *noopCache) Stats() Stats {
	return Stats{
		Misses:     c.stats.misses.Load(),
		Loads:      c.stats.loads.Load(),
		LoadErrors: c.stats.loadErrors.Load(),
	}
}

func (c *noopCache) Ping(_ context.Context) error { return nil }

func (c *noopCache) Close() error { return nil }
