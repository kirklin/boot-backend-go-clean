package usecase

import (
	"context"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/repository"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/usecase"
)

type userUseCase struct {
	userRepo repository.UserRepository
}

func NewUserUseCase(userRepo repository.UserRepository) usecase.UserUseCase {
	return &userUseCase{
		userRepo: userRepo,
	}
}

func (u *userUseCase) GetUserByID(ctx context.Context, id uint) (*entity.User, error) {
	return u.userRepo.FindByID(ctx, id)
}

func (u *userUseCase) UpdateUser(ctx context.Context, user *entity.User) error {
	if err := user.Validate(); err != nil {
		return err
	}

	_, err := u.userRepo.FindByID(ctx, user.ID)
	if err != nil {
		return err
	}

	return u.userRepo.Update(ctx, user)
}

func (u *userUseCase) SoftDeleteUser(ctx context.Context, id uint) error {
	_, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	return u.userRepo.SoftDelete(ctx, id)
}
