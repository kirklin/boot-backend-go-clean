package cache

import (
	"context"
	"errors"
	"time"
)

// GetOrLoadBatch is the batch form of GetOrLoad.
func GetOrLoadBatch[T any](
	ctx context.Context,
	c Cache,
	keys []string,
	ttl time.Duration,
	load func(ctx context.Context, missing []string) (map[string]T, error),
	opts ...LoadOption,
) (map[string]T, error) {
	if load == nil {
		return nil, errors.New("cache: load must not be nil")
	}
	keys = dedupe(keys)
	if len(keys) == 0 {
		return map[string]T{}, nil
	}

	found := make(map[string]T, len(keys))
	missing := make([]string, 0, len(keys))

	entries, err := c.GetBatch(ctx, keys)
	if err != nil {
		entries = nil
	}

	for _, key := range keys {
		entry, cached := entries[key]
		switch {
		case !cached:
			missing = append(missing, key)
		case entry.Missing:
		default:
			var value T
			if err := decode(entry.Payload, &value); err != nil {
				missing = append(missing, key)
				continue
			}
			found[key] = value
		}
	}

	if len(missing) == 0 {
		return found, nil
	}

	loaded, err := load(ctx, missing)
	if err != nil {
		return nil, err
	}

	writes := make(map[string]any, len(loaded))
	absent := make([]string, 0, len(missing))
	for _, key := range missing {
		value, ok := loaded[key]
		if !ok {
			absent = append(absent, key)
			continue
		}
		found[key] = value
		writes[key] = value
	}

	if len(writes) > 0 {
		_ = c.SetBatch(ctx, writes, ttl, opts...)
	}
	if len(absent) > 0 {
		_ = c.SetMissingBatch(ctx, absent, opts...)
	}

	return found, nil
}

// GetBatch is the typed form of Cache.GetBatch.
func GetBatch[T any](ctx context.Context, c Cache, keys []string) (map[string]T, error) {
	entries, err := c.GetBatch(ctx, keys)
	if err != nil {
		return nil, err
	}

	found := make(map[string]T, len(entries))
	for key, entry := range entries {
		if entry.Missing {
			continue
		}
		var value T
		if err := decode(entry.Payload, &value); err != nil {
			continue
		}
		found[key] = value
	}
	return found, nil
}

// dedupe removes repeats while preserving order.
func dedupe(keys []string) []string {
	if len(keys) < 2 {
		return keys
	}

	seen := make(map[string]struct{}, len(keys))
	unique := make([]string, 0, len(keys))
	for _, key := range keys {
		if _, repeated := seen[key]; repeated {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, key)
	}
	return unique
}
