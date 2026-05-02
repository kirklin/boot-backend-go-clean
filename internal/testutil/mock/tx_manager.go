package mock

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/repository"
)

// Compile-time interface conformance check.
var _ repository.TxManager = (*MockTxManager)(nil)

// MockTxManager is a testify mock for repository.TxManager.
//
// By default it simply executes the callback directly (no real transaction).
// Tests can override this behavior via .On("WithTx", ...) expectations.
type MockTxManager struct {
	mock.Mock
}

func (m *MockTxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	args := m.Called(ctx, fn)
	// If no explicit return was set up, default to running fn directly.
	// This makes test setup simpler for most cases: the mock acts as
	// a pass-through, so the test only needs to mock repository calls.
	if args.Get(0) == nil {
		return fn(ctx)
	}
	return args.Error(0)
}

// NewPassthroughTxManager returns a MockTxManager that always executes fn
// directly without any transaction wrapping. This is the default behavior
// expected by most unit tests.
func NewPassthroughTxManager() *MockTxManager {
	m := &MockTxManager{}
	m.On("WithTx", mock.Anything, mock.Anything).Return(nil)
	return m
}
