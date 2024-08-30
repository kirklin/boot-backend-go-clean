package persistence

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/repository"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
)

type userRepository struct {
	db database.Database
}

// NewUserRepository creates a new instance of UserRepository
func NewUserRepository(db database.Database) repository.UserRepository {
	return &userRepository{db: db}
}

// Create inserts a new user into the database
func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	return r.db.DB().WithContext(ctx).Create(user).Error
}

// FindByID retrieves a user by their ID
func (r *userRepository) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	var user entity.User
	err := r.db.DB().WithContext(ctx).First(&user, id).Error
	return r.handleQueryResult(&user, err)
}

// FindByUsername retrieves a user by their username
func (r *userRepository) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	var user entity.User
	err := r.db.DB().WithContext(ctx).Where("username = ?", username).First(&user).Error
	return r.handleQueryResult(&user, err)
}

// FindByEmail retrieves a user by their email
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	err := r.db.DB().WithContext(ctx).Where("email = ?", email).First(&user).Error
	return r.handleQueryResult(&user, err)
}

// Update updates an existing user in the database
func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	result := r.db.DB().WithContext(ctx).Save(user)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrNoRowsAffected
	}
	return nil
}

// SoftDelete marks a user as deleted in the database
func (r *userRepository) SoftDelete(ctx context.Context, id uint) error {
	result := r.db.DB().WithContext(ctx).Delete(&entity.User{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrNoRowsAffected
	}
	return nil
}

// handleQueryResult is a helper function to handle query results
func (r *userRepository) handleQueryResult(user *entity.User, err error) (*entity.User, error) {
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}
