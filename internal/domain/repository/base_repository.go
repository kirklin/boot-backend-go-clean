package repository

import (
	"context"
	"gorm.io/gorm"
)

// BaseRepository defines basic methods to handle transactions.
type BaseRepository interface {
	BeginTx(ctx context.Context) (*gorm.DB, error)
	CommitTx(tx *gorm.DB) error
	RollbackTx(tx *gorm.DB) error
}
