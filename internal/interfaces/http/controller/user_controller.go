package controller

import (
	"context"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/usecase"
	"github.com/kirklin/boot-backend-go-clean/pkg/openapi"
)

type UserController struct {
	userUseCase usecase.UserUseCase
}

func NewUserController(userUseCase usecase.UserUseCase) *UserController {
	return &UserController{
		userUseCase: userUseCase,
	}
}

// ─── Input types ────────────────────────────────────────────────────────────

type GetUserInput struct {
	ID int64 `path:"id"`
}

type UpdateUserInput struct {
	ID   int64       `path:"id"`
	Body entity.User
}

type DeleteUserInput struct {
	ID int64 `path:"id"`
}

// ─── Handlers ───────────────────────────────────────────────────────────────

func (c *UserController) GetUser(ctx context.Context, in *GetUserInput) (*entity.User, error) {
	return c.userUseCase.GetUserByID(ctx, in.ID)
}

func (c *UserController) GetCurrentUser(ctx context.Context, _ *openapi.Empty) (*entity.User, error) {
	userID := openapi.MustUserID(ctx)
	return c.userUseCase.GetUserByID(ctx, userID)
}

func (c *UserController) UpdateUser(ctx context.Context, in *UpdateUserInput) (*openapi.Empty, error) {
	return nil, c.userUseCase.UpdateUser(ctx, &in.Body)
}

func (c *UserController) DeleteUser(ctx context.Context, in *DeleteUserInput) (*openapi.Empty, error) {
	return nil, c.userUseCase.SoftDeleteUser(ctx, in.ID)
}
