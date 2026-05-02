package persistence

import (
	"context"

	"gorm.io/gorm"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/repository"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
)

// gormTxManager implements repository.TxManager using GORM's built-in
// transaction support. It wraps the callback in a database transaction
// and automatically commits or rolls back based on the returned error.
//
// Nested WithTx calls are safe — GORM uses SAVEPOINTs for nested
// transactions, so inner rollbacks don't affect the outer transaction
// unless the outer callback also returns an error.
type gormTxManager struct {
	db database.Database
}

// NewTxManager creates a new TxManager backed by the given database.
func NewTxManager(db database.Database) repository.TxManager {
	return &gormTxManager{db: db}
}

// WithTx executes fn within a database transaction.
//
//   - If fn returns nil, the transaction is committed.
//   - If fn returns an error, the transaction is rolled back and the
//     error is returned to the caller.
//   - If fn panics, GORM's Transaction() recovers, rolls back, and
//     re-panics.
//
// The transaction is injected into ctx via a context key. Repository
// methods that use dbFromContext() will automatically participate in
// this transaction.
func (m *gormTxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.db.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		return fn(txCtx)
	})
}
