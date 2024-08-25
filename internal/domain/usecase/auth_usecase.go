package usecase

import (
	"context"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
)

// AuthUseCase defines the interface for authentication related operations
type AuthUseCase interface {
	// Register handles user registration
	// It takes a RegisterRequest and returns a RegisterResponse with the newly created user's details
	Register(ctx context.Context, req *entity.RegisterRequest) (*entity.RegisterResponse, error)

	// Login authenticates a user and provides access and refresh tokens
	// It takes a LoginRequest and returns a LoginResponse with tokens and user details
	Login(ctx context.Context, req *entity.LoginRequest) (*entity.LoginResponse, error)

	// RefreshToken uses a refresh token to generate new access and refresh tokens
	// It takes a RefreshTokenRequest and returns a RefreshTokenResponse with new tokens
	RefreshToken(ctx context.Context, req *entity.RefreshTokenRequest) (*entity.RefreshTokenResponse, error)

	// Logout invalidates the user's current session
	// It takes a user ID and returns an error if the operation fails
	Logout(refreshToken string, config *configs.AppConfig) error
}
