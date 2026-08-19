package cache

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

type envelope struct {
	Value json.RawMessage `json:"v,omitempty"`

	Missing bool `json:"m,omitempty"`

	StaleAt int64 `json:"s,omitempty"`
}

func (e envelope) isStale(now time.Time) bool {
	return e.StaleAt != 0 && now.UnixMilli() >= e.StaleAt
}

func newEnvelope(value any, now time.Time, staleAfter time.Duration) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("cache: encode value: %w", err)
	}

	wrapped := envelope{Value: encoded}
	if staleAfter > 0 {
		wrapped.StaleAt = now.Add(staleAfter).UnixMilli()
	}
	return json.Marshal(wrapped)
}

func newTombstone() ([]byte, error) {
	return json.Marshal(envelope{Missing: true})
}

func openEnvelope(data []byte) (envelope, error) {
	var wrapped envelope
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return envelope{}, ErrCacheMiss
	}
	if !wrapped.Missing && len(wrapped.Value) == 0 {
		return envelope{}, ErrCacheMiss
	}
	return wrapped, nil
}

func decode(data []byte, dest any) error {
	if err := validateDest(dest); err != nil {
		return err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("cache: decode into %T: %w", dest, err)
	}
	return nil
}

func transfer(value any, dest any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: encode value: %w", err)
	}
	return decode(encoded, dest)
}

func validateDest(dest any) error {
	if dest == nil {
		return fmt.Errorf("cache: dest must be a non-nil pointer, got nil")
	}
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("cache: dest must be a non-nil pointer, got %T", dest)
	}
	return nil
}
