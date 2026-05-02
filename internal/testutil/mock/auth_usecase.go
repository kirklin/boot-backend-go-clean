package mock

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	domainusecase "github.com/kirklin/boot-backend-go-clean/internal/domain/usecase"
)

// Compile-time interface conformance check.
var _ domainusecase.AuthUseCase = (*MockAuthUseCase)(nil)

// MockAuthUseCase is a testify mock for usecase.AuthUseCase.
type MockAuthUseCase struct {
	mock.Mock
}

func (m *MockAuthUseCase) Register(ctx context.Context, req *entity.RegisterRequest) (*entity.RegisterResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.RegisterResponse), args.Error(1)
}

func (m *MockAuthUseCase) Login(ctx context.Context, req *entity.LoginRequest) (*entity.LoginResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.LoginResponse), args.Error(1)
}

func (m *MockAuthUseCase) RefreshToken(ctx context.Context, req *entity.RefreshTokenRequest) (*entity.RefreshTokenResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.RefreshTokenResponse), args.Error(1)
}

func (m *MockAuthUseCase) Logout(ctx context.Context, req *entity.LogoutRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}
