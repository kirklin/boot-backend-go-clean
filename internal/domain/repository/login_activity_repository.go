package repository

import (
	"context"
	"time"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
)

// LoginActivityRepository persists user login activity records.
type LoginActivityRepository interface {
	// Create stores a new login activity record.
	Create(ctx context.Context, activity *entity.LoginActivity) error

	// FindLatestByUserID returns the most recent login record for a user.
	// Returns (nil, nil) if no record is found.
	FindLatestByUserID(ctx context.Context, userID int64) (*entity.LoginActivity, error)

	// DeleteBefore removes all login activity records older than the given time.
	// Returns the number of deleted records.
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
}
