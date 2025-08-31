package persistence

import (
	"context"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/repository"
	"gorm.io/gorm"
)

type baseRepoImpl struct {
	db *gorm.DB
}

// NewBaseRepository creates a new BaseRepository instance.
func NewBaseRepository(db *gorm.DB) repository.BaseRepository {
	return &baseRepoImpl{db: db}
}

// BeginTx Start a new transaction and return the transaction context
func (r *baseRepoImpl) BeginTx(ctx context.Context) (*gorm.DB, error) {
	// 启动事务
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return tx, nil
}

// CommitTx Commit an active transaction
func (r *baseRepoImpl) CommitTx(tx *gorm.DB) error {
	if tx == nil {
		return gorm.ErrInvalidTransaction
	}
	return tx.Commit().Error
}

// RollbackTx Rollback an active transaction
func (r *baseRepoImpl) RollbackTx(tx *gorm.DB) error {
	if tx == nil {
		return gorm.ErrInvalidTransaction
	}
	return tx.Rollback().Error
}
