package persistence

import (
	"context"

	"gorm.io/gorm"

	"github.com/kirklin/boot-backend-go-clean/pkg/database"
)

// txKey is the unexported context key used to store an active *gorm.DB
// transaction. Using a struct type guarantees no collisions with other
// packages' context keys.
type txKey struct{}

// dbFromContext returns the active GORM transaction from ctx if one was
// started by TxManager.WithTx. Otherwise it falls back to the default
// database connection.
//
// This is the ONLY function that repository methods should use to obtain
// a *gorm.DB handle. It transparently participates in transactions without
// requiring any changes to repository method signatures.
func dbFromContext(ctx context.Context, fallback database.Database) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return fallback.DB().WithContext(ctx)
}
