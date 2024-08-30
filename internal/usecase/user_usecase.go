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
	user, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, repository.ErrUserNotFound
	}
	return user, nil
}

func (u *userUseCase) UpdateUser(ctx context.Context, user *entity.User) error {
	if err := user.Validate(); err != nil {
		return err
	}

	existingUser, err := u.userRepo.FindByID(ctx, user.ID)
	if err != nil {
		return err
	}
	if existingUser == nil {
		return repository.ErrUserNotFound
	}
	return u.userRepo.Update(ctx, user)
}

func (u *userUseCase) SoftDeleteUser(ctx context.Context, id uint) error {
	existingUser, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if existingUser == nil {
		return repository.ErrUserNotFound
	}
	return u.userRepo.SoftDelete(ctx, id)
}
