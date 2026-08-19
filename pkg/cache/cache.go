// Package cache is a Redis-backed cache with a no-op implementation for when caching is off
package cache

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrCacheMiss means nothing is cached under the key.
	ErrCacheMiss = errors.New("cache: key not found")

	// ErrNotFound means the origin has no such value; a Loader returns it to record a tombstone.
	ErrNotFound = errors.New("cache: value does not exist at origin")
)

// NoExpiry is what TTL reports for a key with no expiry set.
const NoExpiry time.Duration = -1

// Loader fetches the value for a key after a miss, returning ErrNotFound when
// the origin has nothing.
type Loader func(ctx context.Context) (any, error)

// Cache is a key/value store with per-key expiry. Implementations must be safe
// for concurrent use.
type Cache interface {
	// Get decodes key into dest, a non-nil pointer.
	Get(ctx context.Context, key string, dest any) error

	// Set writes value at key; a ttl of zero means no expiry. TTL jitter is applied.
	Set(ctx context.Context, key string, value any, ttl time.Duration) error

	// SetNX writes value only if the key does not exist. The TTL is exact.
	SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)

	// GetOrLoad decodes key into dest, calling load and caching the result on a miss.
	GetOrLoad(ctx context.Context, key string, dest any, ttl time.Duration, load Loader, opts ...LoadOption) error

	// GetBatch reads several keys in one round trip.
	GetBatch(ctx context.Context, keys []string) (map[string]Entry, error)

	// SetBatch writes several values in one round trip.
	SetBatch(ctx context.Context, values map[string]any, ttl time.Duration, opts ...LoadOption) error

	// SetMissingBatch records tombstones for several keys in one round trip.
	SetMissingBatch(ctx context.Context, keys []string, opts ...LoadOption) error

	// Delete removes the given keys. Write the database first, then Delete.
	Delete(ctx context.Context, keys ...string) error

	// Exists reports whether key is present. A tombstone counts as present.
	Exists(ctx context.Context, key string) (bool, error)

	// Increment adds delta to the integer at key. Counters are raw integers Get cannot read.
	Increment(ctx context.Context, key string, delta int64) (int64, error)

	// Expire sets the TTL of an existing key and reports whether it was there.
	Expire(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// TTL returns the physical time left before key expires
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Stats returns a snapshot of the counters.
	Stats() Stats

	// Ping verifies the cache is reachable.
	Ping(ctx context.Context) error

	// Close releases the underlying connections.
	Close() error
}

// Entry is one result from a batch read.
type Entry struct {
	Payload []byte

	Missing bool
}

// GetOrLoad is the typed form of Cache.GetOrLoad; on ErrNotFound it returns the
// zero value of T.
func GetOrLoad[T any](
	ctx context.Context,
	c Cache,
	key string,
	ttl time.Duration,
	load func(ctx context.Context) (T, error),
	opts ...LoadOption,
) (T, error) {
	var value T
	if load == nil {
		return value, errors.New("cache: load must not be nil")
	}

	err := c.GetOrLoad(ctx, key, &value, ttl, func(ctx context.Context) (any, error) {
		return load(ctx)
	}, opts...)
	if err != nil {
		var zero T
		return zero, err
	}
	return value, nil
}

// Get is the typed form of Cache.Get.
func Get[T any](ctx context.Context, c Cache, key string) (T, error) {
	var value T
	if err := c.Get(ctx, key, &value); err != nil {
		var zero T
		return zero, err
	}
	return value, nil
}

type Stats struct {
	Hits uint64

	Misses uint64

	NegativeHits uint64

	StaleHits uint64

	Loads      uint64
	LoadErrors uint64

	Refreshes uint64

	SharedFlightWaits uint64
}

func (s Stats) HitRate() float64 {
	hits := s.Hits + s.NegativeHits + s.StaleHits
	total := hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}
