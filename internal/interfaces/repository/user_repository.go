package repository

import (
	"context"
	"errors"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/repository"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"

	"gorm.io/gorm"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
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
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// FindByUsername retrieves a user by their username
func (r *userRepository) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	var user entity.User
	err := r.db.DB().WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// FindByEmail retrieves a user by their email
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	err := r.db.DB().WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// Update updates an existing user in the database
func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	return r.db.DB().WithContext(ctx).Save(user).Error
}

// SoftDelete marks a user as deleted in the database
func (r *userRepository) SoftDelete(ctx context.Context, id uint) error {
	return r.db.DB().WithContext(ctx).Delete(&entity.User{}, id).Error
}
