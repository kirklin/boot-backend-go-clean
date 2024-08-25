package usecase

import (
	"context"
	"errors"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/usecase"
	"time"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/repository"
	"github.com/kirklin/boot-backend-go-clean/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

type authUseCase struct {
	userRepo repository.UserRepository
}

func NewAuthUseCase(userRepo repository.UserRepository) usecase.AuthUseCase {
	return &authUseCase{
		userRepo: userRepo,
	}
}

func (a *authUseCase) Register(ctx context.Context, req *entity.RegisterRequest) (*entity.RegisterResponse, error) {
	// Check if user already exists
	existingUser, err := a.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("username already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create new user
	newUser := &entity.User{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	err = a.userRepo.Create(ctx, newUser)
	if err != nil {
		return nil, err
	}

	return &entity.RegisterResponse{User: *newUser}, nil
}

func (a *authUseCase) Login(ctx context.Context, req *entity.LoginRequest) (*entity.LoginResponse, error) {
	user, err := a.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid password")
	}

	// Generate tokens
	tokenPair, err := jwt.GenerateTokenPair(user, 15*time.Minute, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &entity.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
		User:         *user,
	}, nil
}

func (a *authUseCase) RefreshToken(ctx context.Context, req *entity.RefreshTokenRequest) (*entity.RefreshTokenResponse, error) {
	// Validate refresh token
	refreshClaims, _, err := jwt.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, err
	}

	// Get user
	user, err := a.userRepo.FindByID(ctx, refreshClaims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	// Generate new token pair
	tokenPair, err := jwt.GenerateTokenPair(user, 15*time.Minute, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &entity.RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

func (a *authUseCase) Logout(ctx context.Context, userID uint) error {
	// In a real-world scenario, you might want to invalidate the refresh token
	// This could involve storing the token in a blacklist or removing it from storage
	// For simplicity, we'll just return nil here
	return nil
}
