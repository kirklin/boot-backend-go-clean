// Package repository defines the persistence interfaces for domain entities.
package repository

import "context"

// TxManager abstracts database transaction management.
//
// Implementations must guarantee that all repository operations performed
// within the callback fn share the same underlying transaction.
//
//   - If fn returns a non-nil error or panics, the transaction is rolled back.
//   - If fn returns nil, the transaction is committed.
//
// Nested calls to WithTx should be supported via savepoints (GORM default).
//
// Usage in a use case:
//
//	err := txManager.WithTx(ctx, func(txCtx context.Context) error {
//	    if err := repo.Create(txCtx, entity); err != nil {
//	        return err // triggers rollback
//	    }
//	    return repo.Update(txCtx, other) // same transaction
//	})
type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
