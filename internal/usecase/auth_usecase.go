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
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"
)

type authUseCase struct {
	userRepo  repository.UserRepository
	blacklist *auth.TokenBlacklist
	config    *configs.AppConfig
}

func NewAuthUseCase(userRepo repository.UserRepository, blacklist *auth.TokenBlacklist, config *configs.AppConfig) usecase.AuthUseCase {
	return &authUseCase{
		userRepo:  userRepo,
		blacklist: blacklist,
		config:    config,
	}
}

func (a *authUseCase) Register(ctx context.Context, req *entity.RegisterRequest) (*entity.RegisterResponse, error) {
	// Check if user already exists
	existingUser, err := a.userRepo.FindByUsername(ctx, req.Username)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
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
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, repository.ErrUserNotFound
		}
		return nil, err
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid password")
	}

	// Generate tokens
	tokenPair, err := auth.GenerateTokenPair(user)
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
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, repository.ErrUserNotFound
		}
		return nil, err
	}

	// Generate new token pair
	tokenPair, err := auth.GenerateTokenPair(user)
	if err != nil {
		return nil, err
	}

	return &entity.RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

func (a *authUseCase) Logout(ctx context.Context, req *entity.LogoutRequest) error {
	// 检查上下文是否已经取消或超时
	select {
	case <-ctx.Done():
		return ctx.Err() // 返回上下文的错误信息
	default:
		// 将刷新令牌添加到黑名单
		a.blacklist.AddToken(req.RefreshToken, time.Duration(a.config.RefreshTokenLifetime)*time.Hour)
	}

	return nil
}
