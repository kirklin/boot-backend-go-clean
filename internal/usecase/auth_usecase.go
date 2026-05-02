package usecase

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	domainerrors "github.com/kirklin/boot-backend-go-clean/internal/domain/errors"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/gateway"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/repository"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/usecase"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"
)

type authUseCase struct {
	userRepo      repository.UserRepository
	authenticator gateway.Authenticator
	config        *configs.AppConfig
}

func NewAuthUseCase(userRepo repository.UserRepository, authenticator gateway.Authenticator, config *configs.AppConfig) usecase.AuthUseCase {
	return &authUseCase{
		userRepo:      userRepo,
		authenticator: authenticator,
		config:        config,
	}
}

func (a *authUseCase) Register(ctx context.Context, req *entity.RegisterRequest) (*entity.RegisterResponse, error) {
	// Build and validate the user entity (domain-level validation)
	newUser := &entity.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password, // Validated then hashed below
	}
	if err := newUser.Validate(); err != nil {
		return nil, domainerrors.ErrValidationFailed.WithMessage(err.Error())
	}

	// Check if username already exists
	existingUser, err := a.userRepo.FindByUsername(ctx, req.Username)
	if err != nil && !errors.Is(err, domainerrors.ErrUserNotFound) {
		return nil, domainerrors.ErrInternal.Wrap(err)
	}
	if existingUser != nil {
		return nil, domainerrors.ErrUsernameExists
	}

	// Check if email already exists
	existingEmail, err := a.userRepo.FindByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, domainerrors.ErrUserNotFound) {
		return nil, domainerrors.ErrInternal.Wrap(err)
	}
	if existingEmail != nil {
		return nil, domainerrors.ErrEmailExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, domainerrors.ErrInternal.Wrap(err)
	}
	newUser.Password = string(hashedPassword)

	err = a.userRepo.Create(ctx, newUser)
	if err != nil {
		return nil, domainerrors.ErrInternal.Wrap(err)
	}

	return &entity.RegisterResponse{User: *newUser}, nil
}

func (a *authUseCase) Login(ctx context.Context, req *entity.LoginRequest) (*entity.LoginResponse, error) {
	user, err := a.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, domainerrors.ErrUserNotFound) {
			return nil, domainerrors.ErrInvalidCredentials
		}
		return nil, domainerrors.ErrInternal.Wrap(err)
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, domainerrors.ErrInvalidCredentials
	}

	// Generate tokens
	tokenPair, err := a.authenticator.GenerateTokenPair(user)
	if err != nil {
		return nil, domainerrors.ErrInternal.Wrap(err)
	}

	return &entity.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
		User:         *user,
	}, nil
}

func (a *authUseCase) RefreshToken(ctx context.Context, req *entity.RefreshTokenRequest) (*entity.RefreshTokenResponse, error) {
	if a.authenticator.IsTokenBlacklisted(req.RefreshToken) {
		return nil, domainerrors.ErrTokenBlacklisted
	}

	// Validate refresh token
	refreshClaims, _, err := a.authenticator.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, domainerrors.ErrTokenInvalid.Wrap(err)
	}

	// Get user
	user, err := a.userRepo.FindByID(ctx, refreshClaims.UserID)
	if err != nil {
		if errors.Is(err, domainerrors.ErrUserNotFound) {
			return nil, domainerrors.ErrUserNotFound
		}
		return nil, domainerrors.ErrInternal.Wrap(err)
	}

	// Generate new token pair
	tokenPair, err := a.authenticator.GenerateTokenPair(user)
	if err != nil {
		return nil, domainerrors.ErrInternal.Wrap(err)
	}

	// Blacklist old refresh token to prevent replay attacks
	a.authenticator.BlacklistToken(req.RefreshToken, time.Duration(a.config.RefreshTokenLifetime)*time.Hour)

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
		a.authenticator.BlacklistToken(req.RefreshToken, time.Duration(a.config.RefreshTokenLifetime)*time.Hour)
	}

	return nil
}
