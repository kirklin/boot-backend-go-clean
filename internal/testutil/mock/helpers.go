package mock

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// Convenience matchers for common test patterns.
var (
	AnyContext = mock.MatchedBy(func(context.Context) bool { return true })
	AnyFunc    = mock.MatchedBy(func(func(context.Context) error) bool { return true })
)

// NewPassthroughTxManager returns a MockTxManager that executes the callback
// directly without any real transaction wrapping. This is the default behavior
// for most unit tests — the mock simply runs fn(ctx) and returns its error,
// so tests only need to set up repository mock expectations.
//
// The callback's error is correctly propagated: if fn returns
// domainerrors.ErrUsernameExists, that's exactly what WithTx returns.
func NewPassthroughTxManager() *MockTxManager {
	m := &MockTxManager{}
	m.On("WithTx", mock.Anything, mock.Anything).
		Return(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})
	return m
}
