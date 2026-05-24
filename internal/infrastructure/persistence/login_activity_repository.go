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

type loginActivityRepository struct {
	db database.Database
}

// NewLoginActivityRepository creates a new LoginActivityRepository implementation.
func NewLoginActivityRepository(db database.Database) repository.LoginActivityRepository {
	return &loginActivityRepository{db: db}
}

func (r *loginActivityRepository) Create(ctx context.Context, activity *entity.LoginActivity) error {
	dto := &model.LoginActivityDTO{}
	dto.ConvertFromEntity(activity)
	return dbFromContext(ctx, r.db).Create(dto).Error
}

func (r *loginActivityRepository) FindLatestByUserID(ctx context.Context, userID int64) (*entity.LoginActivity, error) {
	var dto model.LoginActivityDTO
	err := dbFromContext(ctx, r.db).
		Where("user_id = ?", userID).
		Order("login_at DESC").
		First(&dto).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No login record yet — not an error
		}
		return nil, err
	}
	return dto.ConvertToEntity(), nil
}
