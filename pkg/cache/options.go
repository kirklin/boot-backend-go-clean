package cache

import "time"

// LoadOption tunes a single GetOrLoad call. Calls that pass none fall back to
// Config.
type LoadOption func(*loadOptions)

type loadOptions struct {
	negativeTTL    time.Duration
	negativeTTLSet bool
	staleGrace     time.Duration
	lockTTL        time.Duration
	refreshTimeout time.Duration
}

// WithNegativeTTL caches "the origin has nothing here" for ttl; zero turns it off.
func WithNegativeTTL(ttl time.Duration) LoadOption {
	return func(o *loadOptions) {
		o.negativeTTL = ttl
		o.negativeTTLSet = true
	}
}

// WithStaleWhileRevalidate serves the cached value for up to grace past its
// logical expiry while refreshing in the background.
func WithStaleWhileRevalidate(grace time.Duration) LoadOption {
	return func(o *loadOptions) { o.staleGrace = grace }
}

// WithSharedFlight collapses the load across replicas with a short Redis lock.
func WithSharedFlight(lockTTL time.Duration) LoadOption {
	return func(o *loadOptions) { o.lockTTL = lockTTL }
}

// WithRefreshTimeout bounds a background refresh.
func WithRefreshTimeout(timeout time.Duration) LoadOption {
	return func(o *loadOptions) { o.refreshTimeout = timeout }
}

func (c *redisCache) resolve(opts []LoadOption) loadOptions {
	resolved := loadOptions{
		negativeTTL:    c.negativeTTL,
		refreshTimeout: c.refreshTimeout,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&resolved)
		}
	}
	if resolved.refreshTimeout <= 0 {
		resolved.refreshTimeout = DefaultRefreshTimeout
	}
	return resolved
}
