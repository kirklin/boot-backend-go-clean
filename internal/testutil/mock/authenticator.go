package mock

import (
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/gateway"
)

// Compile-time interface conformance check.
var _ gateway.Authenticator = (*MockAuthenticator)(nil)

// MockAuthenticator is a testify mock for gateway.Authenticator.
type MockAuthenticator struct {
	mock.Mock
}

func (m *MockAuthenticator) GenerateTokenPair(user *entity.User) (*entity.TokenPair, error) {
	args := m.Called(user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.TokenPair), args.Error(1)
}

func (m *MockAuthenticator) ValidateAccessToken(tokenString string) (*entity.AccessTokenClaims, *entity.StandardClaims, error) {
	args := m.Called(tokenString)
	var ac *entity.AccessTokenClaims
	var sc *entity.StandardClaims
	if args.Get(0) != nil {
		ac = args.Get(0).(*entity.AccessTokenClaims)
	}
	if args.Get(1) != nil {
		sc = args.Get(1).(*entity.StandardClaims)
	}
	return ac, sc, args.Error(2)
}

func (m *MockAuthenticator) ValidateRefreshToken(tokenString string) (*entity.RefreshTokenClaims, *entity.StandardClaims, error) {
	args := m.Called(tokenString)
	var rc *entity.RefreshTokenClaims
	var sc *entity.StandardClaims
	if args.Get(0) != nil {
		rc = args.Get(0).(*entity.RefreshTokenClaims)
	}
	if args.Get(1) != nil {
		sc = args.Get(1).(*entity.StandardClaims)
	}
	return rc, sc, args.Error(2)
}

func (m *MockAuthenticator) BlacklistToken(token string, duration time.Duration) {
	m.Called(token, duration)
}

func (m *MockAuthenticator) IsTokenBlacklisted(token string) bool {
	args := m.Called(token)
	return args.Bool(0)
}
