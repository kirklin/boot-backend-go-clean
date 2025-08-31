package repository

import (
	"context"
	"errors"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByID(ctx context.Context, id int64) (*entity.User, error)
	FindByUsername(ctx context.Context, username string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	SoftDelete(ctx context.Context, id int64) error
}

var (
	ErrUserNotFound = errors.New("user not found")
)
