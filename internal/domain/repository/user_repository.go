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

	// ErrPermissionDenied 表示用户试图执行无权访问的操作或操作属于其他用户的数据。
	ErrPermissionDenied = errors.New("permission denied for this operation")
)
