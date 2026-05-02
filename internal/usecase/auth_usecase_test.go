package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	domainerrors "github.com/kirklin/boot-backend-go-clean/internal/domain/errors"
	testmock "github.com/kirklin/boot-backend-go-clean/internal/testutil/mock"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"
)

func newAuthUseCase(repo *testmock.MockUserRepository, auth *testmock.MockAuthenticator) *authUseCase {
	return &authUseCase{
		userRepo:      repo,
		authenticator: auth,
		config:        &configs.AppConfig{RefreshTokenLifetime: 24},
	}
}

// ─── Register ─────────────────────────────────────────────────────────────────

func TestAuthUseCase_Register_Success(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	repo.On("FindByUsername", mock.Anything, "newuser").Return(nil, domainerrors.ErrUserNotFound)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).Return(nil)

	resp, err := uc.Register(context.Background(), &entity.RegisterRequest{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "securepassword",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "newuser", resp.User.Username)
	repo.AssertExpectations(t)
}

func TestAuthUseCase_Register_UsernameExists(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	existingUser := &entity.User{ID: 1, Username: "existing"}
	repo.On("FindByUsername", mock.Anything, "existing").Return(existingUser, nil)

	resp, err := uc.Register(context.Background(), &entity.RegisterRequest{
		Username: "existing",
		Email:    "test@example.com",
		Password: "securepassword",
	})

	assert.ErrorIs(t, err, domainerrors.ErrUsernameExists)
	assert.Nil(t, resp)
}

// ─── Login ────────────────────────────────────────────────────────────────────

func TestAuthUseCase_Login_Success(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	// bcrypt hash of "correctpassword"
	user := &entity.User{
		ID:       1,
		Username: "kirk",
	}
	// We need a real bcrypt hash for comparison
	hashedPw, _ := bcryptHash("correctpassword")
	user.Password = hashedPw

	repo.On("FindByUsername", mock.Anything, "kirk").Return(user, nil)
	auth.On("GenerateTokenPair", user).Return(&entity.TokenPair{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}, nil)

	resp, err := uc.Login(context.Background(), &entity.LoginRequest{
		Username: "kirk",
		Password: "correctpassword",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "access-token", resp.AccessToken)
	repo.AssertExpectations(t)
	auth.AssertExpectations(t)
}

func TestAuthUseCase_Login_UserNotFound(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	repo.On("FindByUsername", mock.Anything, "ghost").Return(nil, domainerrors.ErrUserNotFound)

	resp, err := uc.Login(context.Background(), &entity.LoginRequest{
		Username: "ghost",
		Password: "whatever",
	})

	assert.ErrorIs(t, err, domainerrors.ErrInvalidCredentials)
	assert.Nil(t, resp)
}

func TestAuthUseCase_Login_WrongPassword(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	hashedPw, _ := bcryptHash("correctpassword")
	user := &entity.User{ID: 1, Username: "kirk", Password: hashedPw}
	repo.On("FindByUsername", mock.Anything, "kirk").Return(user, nil)

	resp, err := uc.Login(context.Background(), &entity.LoginRequest{
		Username: "kirk",
		Password: "wrongpassword",
	})

	assert.ErrorIs(t, err, domainerrors.ErrInvalidCredentials)
	assert.Nil(t, resp)
}

// ─── RefreshToken ─────────────────────────────────────────────────────────────

func TestAuthUseCase_RefreshToken_Success(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	auth.On("IsTokenBlacklisted", "valid-refresh").Return(false)
	auth.On("ValidateRefreshToken", "valid-refresh").Return(
		&entity.RefreshTokenClaims{UserID: 1},
		&entity.StandardClaims{},
		nil,
	)

	user := &entity.User{ID: 1, Username: "kirk"}
	repo.On("FindByID", mock.Anything, int64(1)).Return(user, nil)
	auth.On("GenerateTokenPair", user).Return(&entity.TokenPair{
		AccessToken:  "new-access",
		RefreshToken: "new-refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
	}, nil)

	resp, err := uc.RefreshToken(context.Background(), &entity.RefreshTokenRequest{
		RefreshToken: "valid-refresh",
	})

	assert.NoError(t, err)
	assert.Equal(t, "new-access", resp.AccessToken)
	auth.AssertExpectations(t)
}

func TestAuthUseCase_RefreshToken_Blacklisted(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	auth.On("IsTokenBlacklisted", "bad-token").Return(true)

	resp, err := uc.RefreshToken(context.Background(), &entity.RefreshTokenRequest{
		RefreshToken: "bad-token",
	})

	assert.ErrorIs(t, err, domainerrors.ErrTokenBlacklisted)
	assert.Nil(t, resp)
}

// ─── Logout ───────────────────────────────────────────────────────────────────

func TestAuthUseCase_Logout_Success(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	auth.On("BlacklistToken", "token-to-revoke", 24*time.Hour).Return()

	err := uc.Logout(context.Background(), &entity.LogoutRequest{
		RefreshToken: "token-to-revoke",
	})

	assert.NoError(t, err)
	auth.AssertExpectations(t)
}

func TestAuthUseCase_Logout_CancelledContext(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := uc.Logout(ctx, &entity.LogoutRequest{RefreshToken: "token"})

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// ─── Register Error Branches ──────────────────────────────────────────────────

func TestAuthUseCase_Register_DBErrorOnFindByUsername(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	dbErr := fmt.Errorf("connection refused")
	repo.On("FindByUsername", mock.Anything, "newuser").Return(nil, domainerrors.ErrInternal.Wrap(dbErr))

	resp, err := uc.Register(context.Background(), &entity.RegisterRequest{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "securepassword",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	// Should wrap as internal error, not expose DB details
	var appErr *domainerrors.AppError
	assert.True(t, errors.As(err, &appErr))
}

func TestAuthUseCase_Register_CreateFails(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	repo.On("FindByUsername", mock.Anything, "newuser").Return(nil, domainerrors.ErrUserNotFound)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).Return(fmt.Errorf("unique constraint violated"))

	resp, err := uc.Register(context.Background(), &entity.RegisterRequest{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "securepassword",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// ─── Login Error Branches ─────────────────────────────────────────────────────

func TestAuthUseCase_Login_DBError(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	// DB returns a non-ErrUserNotFound error
	repo.On("FindByUsername", mock.Anything, "kirk").Return(nil, domainerrors.ErrInternal.Wrap(fmt.Errorf("db down")))

	resp, err := uc.Login(context.Background(), &entity.LoginRequest{
		Username: "kirk",
		Password: "whatever",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	// Should NOT be ErrInvalidCredentials — it's a real internal error
	assert.False(t, errors.Is(err, domainerrors.ErrInvalidCredentials))
}

func TestAuthUseCase_Login_GenerateTokenPairFails(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	hashedPw, _ := bcryptHash("correctpassword")
	user := &entity.User{ID: 1, Username: "kirk", Password: hashedPw}
	repo.On("FindByUsername", mock.Anything, "kirk").Return(user, nil)
	auth.On("GenerateTokenPair", user).Return(nil, fmt.Errorf("signing failure"))

	resp, err := uc.Login(context.Background(), &entity.LoginRequest{
		Username: "kirk",
		Password: "correctpassword",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// ─── RefreshToken Error Branches ──────────────────────────────────────────────

func TestAuthUseCase_RefreshToken_ValidationFails(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	auth.On("IsTokenBlacklisted", "tampered-token").Return(false)
	auth.On("ValidateRefreshToken", "tampered-token").Return(nil, nil, fmt.Errorf("invalid signature"))

	resp, err := uc.RefreshToken(context.Background(), &entity.RefreshTokenRequest{
		RefreshToken: "tampered-token",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	var appErr *domainerrors.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, "TOKEN_INVALID", appErr.Code)
}

func TestAuthUseCase_RefreshToken_UserNotFound(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	auth.On("IsTokenBlacklisted", "valid-refresh").Return(false)
	auth.On("ValidateRefreshToken", "valid-refresh").Return(
		&entity.RefreshTokenClaims{UserID: 999},
		&entity.StandardClaims{},
		nil,
	)
	repo.On("FindByID", mock.Anything, int64(999)).Return(nil, domainerrors.ErrUserNotFound)

	resp, err := uc.RefreshToken(context.Background(), &entity.RefreshTokenRequest{
		RefreshToken: "valid-refresh",
	})

	assert.ErrorIs(t, err, domainerrors.ErrUserNotFound)
	assert.Nil(t, resp)
}

func TestAuthUseCase_RefreshToken_DBError(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	auth.On("IsTokenBlacklisted", "valid-refresh").Return(false)
	auth.On("ValidateRefreshToken", "valid-refresh").Return(
		&entity.RefreshTokenClaims{UserID: 1},
		&entity.StandardClaims{},
		nil,
	)
	repo.On("FindByID", mock.Anything, int64(1)).Return(nil, domainerrors.ErrInternal.Wrap(fmt.Errorf("db timeout")))

	resp, err := uc.RefreshToken(context.Background(), &entity.RefreshTokenRequest{
		RefreshToken: "valid-refresh",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.False(t, errors.Is(err, domainerrors.ErrUserNotFound))
}

func TestAuthUseCase_RefreshToken_GenerateTokenPairFails(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	auth.On("IsTokenBlacklisted", "valid-refresh").Return(false)
	auth.On("ValidateRefreshToken", "valid-refresh").Return(
		&entity.RefreshTokenClaims{UserID: 1},
		&entity.StandardClaims{},
		nil,
	)
	user := &entity.User{ID: 1, Username: "kirk"}
	repo.On("FindByID", mock.Anything, int64(1)).Return(user, nil)
	auth.On("GenerateTokenPair", user).Return(nil, fmt.Errorf("key rotation in progress"))

	resp, err := uc.RefreshToken(context.Background(), &entity.RefreshTokenRequest{
		RefreshToken: "valid-refresh",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func bcryptHash(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashed), err
}
