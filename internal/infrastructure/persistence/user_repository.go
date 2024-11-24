package persistence

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/repository"
	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/persistence/model"
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
	dto := model.UserDTO{}
	dto.ConvertFromEntity(user)

	err := r.db.DB().WithContext(ctx).Create(&dto).Error
	if err != nil {
		return err
	}

	// 在创建成功后，把 DTO 的 ID 同步回领域实体
	*user = *dto.ConvertToEntity()
	return nil
}

// FindByID retrieves a user by their ID
func (r *userRepository) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	var dto model.UserDTO
	err := r.db.DB().WithContext(ctx).First(&dto, id).Error
	return r.handleQueryResult(&dto, err)
}

// FindByUsername retrieves a user by their username
func (r *userRepository) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	var dto model.UserDTO
	err := r.db.DB().WithContext(ctx).Where("username = ?", username).First(&dto).Error
	return r.handleQueryResult(&dto, err)
}

// FindByEmail retrieves a user by their email
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var dto model.UserDTO
	err := r.db.DB().WithContext(ctx).Where("email = ?", email).First(&dto).Error
	return r.handleQueryResult(&dto, err)
}

// Update updates an existing user in the database
func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	var dto model.UserDTO
	dto.ConvertFromEntity(user)
	result := r.db.DB().WithContext(ctx).Save(&dto)
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
	result := r.db.DB().WithContext(ctx).Delete(&model.UserDTO{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrNoRowsAffected
	}
	return nil
}

func (r *userRepository) handleQueryResult(dto *model.UserDTO, err error) (*entity.User, error) {
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrUserNotFound
		}
		return nil, err
	}
	return dto.ConvertToEntity(), nil
}
