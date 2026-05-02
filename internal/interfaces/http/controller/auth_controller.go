package controller

import (
	"context"
	"net/http"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity/response"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/usecase"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/humaerr"
)

type AuthController struct {
	authUseCase usecase.AuthUseCase
}

func NewAuthController(authUseCase usecase.AuthUseCase) *AuthController {
	return &AuthController{
		authUseCase: authUseCase,
	}
}

// --- Huma Input/Output types ---

type RegisterInput struct {
	Body entity.RegisterRequest
}

type RegisterOutput struct {
	Body response.Response[*entity.RegisterResponse]
}

type LoginInput struct {
	Body entity.LoginRequest
}

type LoginOutput struct {
	Body response.Response[*entity.LoginResponse]
}

type RefreshTokenInput struct {
	Body entity.RefreshTokenRequest
}

type RefreshTokenOutput struct {
	Body response.Response[*entity.RefreshTokenResponse]
}

type LogoutInput struct {
	Body entity.LogoutRequest
}

type LogoutOutput struct {
	Body response.Response[any]
}

// --- Huma handlers ---

func (c *AuthController) Register(ctx context.Context, input *RegisterInput) (*RegisterOutput, error) {
	resp, err := c.authUseCase.Register(ctx, &input.Body)
	if err != nil {
		return nil, humaerr.NewHumaError(http.StatusInternalServerError, "Registration failed", err)
	}

	return &RegisterOutput{
		Body: response.NewSuccessResponse("User registered successfully", resp),
	}, nil
}

func (c *AuthController) Login(ctx context.Context, input *LoginInput) (*LoginOutput, error) {
	resp, err := c.authUseCase.Login(ctx, &input.Body)
	if err != nil {
		return nil, humaerr.NewHumaError(http.StatusUnauthorized, "Login failed", err)
	}

	return &LoginOutput{
		Body: response.NewSuccessResponse("Login successful", resp),
	}, nil
}

func (c *AuthController) RefreshToken(ctx context.Context, input *RefreshTokenInput) (*RefreshTokenOutput, error) {
	resp, err := c.authUseCase.RefreshToken(ctx, &input.Body)
	if err != nil {
		return nil, humaerr.NewHumaError(http.StatusUnauthorized, "Token refresh failed", err)
	}

	return &RefreshTokenOutput{
		Body: response.NewSuccessResponse("Token refreshed successfully", resp),
	}, nil
}

func (c *AuthController) Logout(ctx context.Context, input *LogoutInput) (*LogoutOutput, error) {
	err := c.authUseCase.Logout(ctx, &input.Body)
	if err != nil {
		return nil, humaerr.NewHumaError(http.StatusInternalServerError, "Logout failed", err)
	}

	return &LogoutOutput{
		Body: response.NewSuccessResponse[any]("Logged out successfully", nil),
	}, nil
}
