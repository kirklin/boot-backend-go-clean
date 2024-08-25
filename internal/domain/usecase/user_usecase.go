package usecase

import (
	"context"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
)

// UserUseCase defines the interface for user-related operations
type UserUseCase interface {
	// GetUserByID retrieves a user by their ID
	// It takes a user ID and returns the corresponding User entity
	GetUserByID(ctx context.Context, id uint) (*entity.User, error)

	// UpdateUser updates the information of an existing user
	// It takes a User entity with updated information and returns an error if the operation fails
	UpdateUser(ctx context.Context, user *entity.User) error

	// SoftDeleteUser marks a user as deleted in the system
	// It takes a user ID and returns an error if the operation fails
	SoftDeleteUser(ctx context.Context, id uint) error
}
