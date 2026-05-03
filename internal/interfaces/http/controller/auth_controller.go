package controller

import (
	"context"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/usecase"
	"github.com/kirklin/boot-backend-go-clean/pkg/openapi"
)

type AuthController struct {
	authUseCase usecase.AuthUseCase
}

func NewAuthController(authUseCase usecase.AuthUseCase) *AuthController {
	return &AuthController{
		authUseCase: authUseCase,
	}
}

// ─── Input types ────────────────────────────────────────────────────────────

type RegisterInput struct {
	Body entity.RegisterRequest
}

type LoginInput struct {
	Body entity.LoginRequest
}

type RefreshInput struct {
	Body entity.RefreshTokenRequest
}

type LogoutInput struct {
	Body entity.LogoutRequest
}

// ─── Handlers ───────────────────────────────────────────────────────────────

func (c *AuthController) Register(ctx context.Context, in *RegisterInput) (*entity.RegisterResponse, error) {
	return c.authUseCase.Register(ctx, &in.Body)
}

func (c *AuthController) Login(ctx context.Context, in *LoginInput) (*entity.LoginResponse, error) {
	return c.authUseCase.Login(ctx, &in.Body)
}

func (c *AuthController) RefreshToken(ctx context.Context, in *RefreshInput) (*entity.RefreshTokenResponse, error) {
	return c.authUseCase.RefreshToken(ctx, &in.Body)
}

func (c *AuthController) Logout(ctx context.Context, in *LogoutInput) (*openapi.Empty, error) {
	return nil, c.authUseCase.Logout(ctx, &in.Body)
}
