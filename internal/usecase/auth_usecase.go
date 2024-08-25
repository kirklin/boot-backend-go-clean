package usecase

import (
	"context"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"time"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/repository"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/usecase"
	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/auth"
)

type authUseCase struct {
	userRepo  repository.UserRepository
	blacklist *auth.TokenBlacklist
}

func NewAuthUseCase(userRepo repository.UserRepository, blacklist *auth.TokenBlacklist) usecase.AuthUseCase {
	return &authUseCase{
		userRepo:  userRepo,
		blacklist: blacklist,
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
	tokenPair, err := auth.GenerateTokenPair(user, 15*time.Minute, 7*24*time.Hour)
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

	if a.blacklist.IsTokenBlacklisted(req.RefreshToken) {
		return nil, errors.New("refresh token is blacklisted")
	}

	// Validate refresh token
	refreshClaims, _, err := auth.ValidateRefreshToken(req.RefreshToken)
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
	tokenPair, err := auth.GenerateTokenPair(user, 15*time.Minute, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &entity.RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

func (a *authUseCase) Logout(ctx context.Context, refreshToken string) error {
	// 将刷新令牌添加到黑名单，设置过期时间为 7 天
	a.blacklist.AddToken(refreshToken, 7*24*time.Hour)
	return nil
}
