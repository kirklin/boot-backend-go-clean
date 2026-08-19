package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	mathrand "math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

type Config struct {
	Addr string

	Password string

	DB int

	PoolSize int

	DialTimeout time.Duration

	ReadTimeout time.Duration

	WriteTimeout time.Duration

	KeyPrefix string

	// TTLJitter spreads expiry times as a fraction of the TTL, to prevent
	// avalanches.
	TTLJitter float64

	// NegativeTTL is the default lifetime of a tombstone; zero disables negative
	NegativeTTL time.Duration

	// RefreshTimeout bounds a background refresh.
	RefreshTimeout time.Duration

	OnError func(key string, err error)
}

const (
	DefaultDialTimeout    = 5 * time.Second
	DefaultCommandTimeout = 3 * time.Second
	DefaultRefreshTimeout = 10 * time.Second

	sharedFlightPollInterval = 25 * time.Millisecond
)

var errFlightElsewhere = errors.New("cache: load already in flight elsewhere")

var releaseLockScript = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0
`)

type counters struct {
	hits              atomic.Uint64
	misses            atomic.Uint64
	negativeHits      atomic.Uint64
	staleHits         atomic.Uint64
	loads             atomic.Uint64
	loadErrors        atomic.Uint64
	refreshes         atomic.Uint64
	sharedFlightWaits atomic.Uint64
}

type redisCache struct {
	client         *redis.Client
	prefix         string
	jitter         float64
	negativeTTL    time.Duration
	refreshTimeout time.Duration
	onError        func(key string, err error)

	group singleflight.Group

	refreshing sync.Map

	stats counters

	now func() time.Time
}

// NewRedis connects to Redis and returns a ready Cache, pinging before it
// returns.
func NewRedis(cfg Config) (Cache, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	dialTimeout := orDefault(cfg.DialTimeout, DefaultDialTimeout)
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  dialTimeout,
		ReadTimeout:  orDefault(cfg.ReadTimeout, DefaultCommandTimeout),
		WriteTimeout: orDefault(cfg.WriteTimeout, DefaultCommandTimeout),
	})

	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("cache: connect to redis at %s: %w", cfg.Addr, err)
	}

	return NewRedisFromClient(client, cfg), nil
}

func NewRedisFromClient(client *redis.Client, cfg Config) Cache {
	return &redisCache{
		client:         client,
		prefix:         normalizePrefix(cfg.KeyPrefix),
		jitter:         cfg.TTLJitter,
		negativeTTL:    cfg.NegativeTTL,
		refreshTimeout: orDefault(cfg.RefreshTimeout, DefaultRefreshTimeout),
		onError:        cfg.OnError,
		now:            time.Now,
	}
}

func (cfg Config) validate() error {
	if cfg.Addr == "" {
		return errors.New("cache: Addr is required")
	}
	if cfg.TTLJitter < 0 || cfg.TTLJitter >= 1 {
		return fmt.Errorf("cache: TTLJitter must be in [0, 1), got %v", cfg.TTLJitter)
	}
	if cfg.NegativeTTL < 0 {
		return fmt.Errorf("cache: NegativeTTL must not be negative, got %v", cfg.NegativeTTL)
	}
	return nil
}

func orDefault(value, fallback time.Duration) time.Duration {
	if value <= 0 {
		return fallback
	}
	return value
}

func normalizePrefix(prefix string) string {
	if prefix == "" || prefix[len(prefix)-1] == ':' {
		return prefix
	}
	return prefix + ":"
}

func (c *redisCache) key(key string) string     { return c.prefix + key }
func (c *redisCache) lockKey(key string) string { return c.prefix + "lock:" + key }

func (c *redisCache) reportError(key string, err error) {
	if c.onError != nil && err != nil {
		c.onError(key, err)
	}
}

func (c *redisCache) jittered(ttl time.Duration) time.Duration {
	if ttl <= 0 || c.jitter <= 0 {
		return ttl
	}

	offset := (mathrand.Float64()*2 - 1) * c.jitter
	spread := time.Duration(float64(ttl) * (1 + offset))
	if spread <= 0 {
		return ttl
	}
	return spread
}

func (c *redisCache) readEnvelope(ctx context.Context, key string) (envelope, error) {
	data, err := c.client.Get(ctx, c.key(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return envelope{}, ErrCacheMiss
	}
	if err != nil {
		return envelope{}, fmt.Errorf("cache: get %q: %w", key, err)
	}
	return openEnvelope(data)
}

func (c *redisCache) Get(ctx context.Context, key string, dest any) error {
	if err := validateDest(dest); err != nil {
		return err
	}

	entry, err := c.readEnvelope(ctx, key)
	if err != nil {
		return err
	}
	if entry.Missing {
		return ErrNotFound
	}
	return decode(entry.Value, dest)
}

func (c *redisCache) GetOrLoad(
	ctx context.Context,
	key string,
	dest any,
	ttl time.Duration,
	load Loader,
	opts ...LoadOption,
) error {
	if err := validateDest(dest); err != nil {
		return err
	}
	if load == nil {
		return errors.New("cache: load must not be nil")
	}
	options := c.resolve(opts)

	if entry, err := c.readEnvelope(ctx, key); err == nil {
		switch {
		case entry.Missing:
			c.stats.negativeHits.Add(1)
			return ErrNotFound

		case !entry.isStale(c.now()):
			c.stats.hits.Add(1)
			return decode(entry.Value, dest)

		case options.staleGrace > 0:
			c.stats.staleHits.Add(1)
			c.refreshInBackground(ctx, key, ttl, load, options)
			return decode(entry.Value, dest)
		}
	}

	c.stats.misses.Add(1)
	payload, err := c.loadAndStore(ctx, key, ttl, load, options, true)
	if err != nil {
		return err
	}
	return decode(payload, dest)
}

func (c *redisCache) loadAndStore(
	ctx context.Context,
	key string,
	ttl time.Duration,
	load Loader,
	options loadOptions,
	wait bool,
) ([]byte, error) {
	result, err, _ := c.group.Do(key, func() (any, error) {
		if entry, err := c.readEnvelope(ctx, key); err == nil {
			if entry.Missing {
				return nil, ErrNotFound
			}
			if !entry.isStale(c.now()) {
				return []byte(entry.Value), nil
			}
		}

		if options.lockTTL > 0 {
			token, acquired, lockErr := c.acquireLock(ctx, key, options.lockTTL)
			switch {
			case lockErr != nil:

			case acquired:

				defer c.releaseLock(key, token)

			default:
				c.stats.sharedFlightWaits.Add(1)
				if !wait {
					return nil, errFlightElsewhere
				}
				if filled, ok := c.waitForFill(ctx, key, options); ok {
					return filled, nil
				}
			}
		}

		return c.callLoader(ctx, key, ttl, load, options)
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrCacheMiss
	}

	payload, ok := result.([]byte)
	if !ok {
		return nil, fmt.Errorf("cache: loader for %q produced %T, want []byte", key, result)
	}
	return payload, nil
}

func (c *redisCache) callLoader(
	ctx context.Context,
	key string,
	ttl time.Duration,
	load Loader,
	options loadOptions,
) ([]byte, error) {
	c.stats.loads.Add(1)

	value, err := load(ctx)
	if errors.Is(err, ErrNotFound) {
		c.storeTombstone(ctx, key, options)
		return nil, ErrNotFound
	}
	if err != nil {
		c.stats.loadErrors.Add(1)
		return nil, err
	}

	fresh := c.jittered(ttl)
	staleAfter := time.Duration(0)
	physical := fresh
	if options.staleGrace > 0 {
		staleAfter = fresh
		physical = fresh + options.staleGrace
	}

	encoded, err := newEnvelope(value, c.now(), staleAfter)
	if err != nil {
		return nil, err
	}
	if err := c.client.Set(ctx, c.key(key), encoded, physical).Err(); err != nil {
		c.reportError(key, fmt.Errorf("cache: write back %q: %w", key, err))
	}

	entry, err := openEnvelope(encoded)
	if err != nil {
		return nil, err
	}
	return entry.Value, nil
}

func (c *redisCache) storeTombstone(ctx context.Context, key string, options loadOptions) {
	ttl := options.negativeTTL
	if !options.negativeTTLSet {
		ttl = c.negativeTTL
	}
	if ttl <= 0 {
		return
	}

	encoded, err := newTombstone()
	if err != nil {
		c.reportError(key, err)
		return
	}
	if err := c.client.Set(ctx, c.key(key), encoded, c.jittered(ttl)).Err(); err != nil {
		c.reportError(key, fmt.Errorf("cache: write tombstone %q: %w", key, err))
	}
}

func (c *redisCache) refreshInBackground(
	ctx context.Context,
	key string,
	ttl time.Duration,
	load Loader,
	options loadOptions,
) {
	if _, busy := c.refreshing.LoadOrStore(key, struct{}{}); busy {
		return
	}
	c.stats.refreshes.Add(1)

	refreshCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), options.refreshTimeout)

	go func() {
		defer cancel()
		defer c.refreshing.Delete(key)

		_, err := c.loadAndStore(refreshCtx, key, ttl, load, options, false)
		switch {
		case err == nil, errors.Is(err, ErrNotFound), errors.Is(err, errFlightElsewhere):
			return
		default:
			c.reportError(key, fmt.Errorf("cache: background refresh %q: %w", key, err))
		}
	}()
}

func (c *redisCache) waitForFill(ctx context.Context, key string, options loadOptions) ([]byte, bool) {
	ticker := time.NewTicker(sharedFlightPollInterval)
	defer ticker.Stop()

	deadline := time.After(options.lockTTL)
	for {
		select {
		case <-ctx.Done():
			return nil, false
		case <-deadline:
			return nil, false
		case <-ticker.C:
			entry, err := c.readEnvelope(ctx, key)
			if err != nil {
				continue
			}
			if entry.Missing {
				return nil, false
			}
			if !entry.isStale(c.now()) {
				return entry.Value, true
			}
		}
	}
}

func (c *redisCache) acquireLock(ctx context.Context, key string, ttl time.Duration) (string, bool, error) {
	token, err := newLockToken()
	if err != nil {
		return "", false, err
	}

	err = c.client.SetArgs(ctx, c.lockKey(key), token, redis.SetArgs{Mode: "NX", TTL: ttl}).Err()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("cache: acquire lock %q: %w", key, err)
	}
	return token, true, nil
}

func (c *redisCache) releaseLock(key, token string) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultCommandTimeout)
	defer cancel()

	if err := releaseLockScript.Run(ctx, c.client, []string{c.lockKey(key)}, token).Err(); err != nil && !errors.Is(err, redis.Nil) {
		c.reportError(key, fmt.Errorf("cache: release lock %q: %w", key, err))
	}
}

func newLockToken() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("cache: generate lock token: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

func (c *redisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	encoded, err := newEnvelope(value, c.now(), 0)
	if err != nil {
		return err
	}
	if err := c.client.Set(ctx, c.key(key), encoded, c.jittered(ttl)).Err(); err != nil {
		return fmt.Errorf("cache: set %q: %w", key, err)
	}
	return nil
}

func (c *redisCache) SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	encoded, err := newEnvelope(value, c.now(), 0)
	if err != nil {
		return false, err
	}

	err = c.client.SetArgs(ctx, c.key(key), encoded, redis.SetArgs{Mode: "NX", TTL: ttl}).Err()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cache: setnx %q: %w", key, err)
	}
	return true, nil
}

func (c *redisCache) GetBatch(ctx context.Context, keys []string) (map[string]Entry, error) {
	if len(keys) == 0 {
		return map[string]Entry{}, nil
	}

	prefixed := make([]string, len(keys))
	for i, key := range keys {
		prefixed[i] = c.key(key)
	}

	values, err := c.client.MGet(ctx, prefixed...).Result()
	if err != nil {
		return nil, fmt.Errorf("cache: get %d key(s): %w", len(keys), err)
	}

	now := c.now()
	entries := make(map[string]Entry, len(keys))
	for i, raw := range values {
		if i >= len(keys) {
			break
		}
		key := keys[i]

		text, ok := raw.(string)
		if !ok {
			c.stats.misses.Add(1)
			continue
		}

		entry, err := openEnvelope([]byte(text))
		if err != nil {
			c.stats.misses.Add(1)
			continue
		}
		if entry.Missing {
			c.stats.negativeHits.Add(1)
			entries[key] = Entry{Missing: true}
			continue
		}
		if entry.isStale(now) {
			c.stats.misses.Add(1)
			continue
		}

		c.stats.hits.Add(1)
		entries[key] = Entry{Payload: entry.Value}
	}
	return entries, nil
}

func (c *redisCache) SetBatch(ctx context.Context, values map[string]any, ttl time.Duration, opts ...LoadOption) error {
	if len(values) == 0 {
		return nil
	}
	options := c.resolve(opts)
	now := c.now()

	pipe := c.client.Pipeline()
	for key, value := range values {
		fresh := c.jittered(ttl)
		staleAfter := time.Duration(0)
		physical := fresh
		if options.staleGrace > 0 {
			staleAfter = fresh
			physical = fresh + options.staleGrace
		}

		encoded, err := newEnvelope(value, now, staleAfter)
		if err != nil {
			return err
		}
		pipe.Set(ctx, c.key(key), encoded, physical)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		wrapped := fmt.Errorf("cache: set %d key(s): %w", len(values), err)
		c.reportError("", wrapped)
		return wrapped
	}
	return nil
}

func (c *redisCache) SetMissingBatch(ctx context.Context, keys []string, opts ...LoadOption) error {
	if len(keys) == 0 {
		return nil
	}
	options := c.resolve(opts)

	ttl := options.negativeTTL
	if !options.negativeTTLSet {
		ttl = c.negativeTTL
	}
	if ttl <= 0 {
		return nil
	}

	encoded, err := newTombstone()
	if err != nil {
		return err
	}

	pipe := c.client.Pipeline()
	for _, key := range keys {
		pipe.Set(ctx, c.key(key), encoded, c.jittered(ttl))
	}

	if _, err := pipe.Exec(ctx); err != nil {
		wrapped := fmt.Errorf("cache: set %d tombstone(s): %w", len(keys), err)
		c.reportError("", wrapped)
		return wrapped
	}
	return nil
}

func (c *redisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	prefixed := make([]string, len(keys))
	for i, key := range keys {
		prefixed[i] = c.key(key)
	}
	if err := c.client.Del(ctx, prefixed...).Err(); err != nil {
		return fmt.Errorf("cache: delete %d key(s): %w", len(keys), err)
	}
	return nil
}

func (c *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, c.key(key)).Result()
	if err != nil {
		return false, fmt.Errorf("cache: exists %q: %w", key, err)
	}
	return count > 0, nil
}

func (c *redisCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	value, err := c.client.IncrBy(ctx, c.key(key), delta).Result()
	if err != nil {
		return 0, fmt.Errorf("cache: increment %q: %w", key, err)
	}
	return value, nil
}

func (c *redisCache) Expire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	ok, err := c.client.Expire(ctx, c.key(key), ttl).Result()
	if err != nil {
		return false, fmt.Errorf("cache: expire %q: %w", key, err)
	}
	return ok, nil
}

func (c *redisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := c.client.TTL(ctx, c.key(key)).Result()
	if err != nil {
		return 0, fmt.Errorf("cache: ttl %q: %w", key, err)
	}

	switch ttl {
	case -2:
		return 0, ErrCacheMiss
	case -1:
		return NoExpiry, nil
	}
	return ttl, nil
}

func (c *redisCache) Stats() Stats {
	return Stats{
		Hits:              c.stats.hits.Load(),
		Misses:            c.stats.misses.Load(),
		NegativeHits:      c.stats.negativeHits.Load(),
		StaleHits:         c.stats.staleHits.Load(),
		Loads:             c.stats.loads.Load(),
		LoadErrors:        c.stats.loadErrors.Load(),
		Refreshes:         c.stats.refreshes.Load(),
		SharedFlightWaits: c.stats.sharedFlightWaits.Load(),
	}
}

func (c *redisCache) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("cache: ping: %w", err)
	}
	return nil
}

func (c *redisCache) Close() error {
	return c.client.Close()
}
