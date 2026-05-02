package mock

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	domainusecase "github.com/kirklin/boot-backend-go-clean/internal/domain/usecase"
)

// Compile-time interface conformance check.
var _ domainusecase.UserUseCase = (*MockUserUseCase)(nil)

// MockUserUseCase is a testify mock for usecase.UserUseCase.
type MockUserUseCase struct {
	mock.Mock
}

func (m *MockUserUseCase) GetUserByID(ctx context.Context, id int64) (*entity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserUseCase) UpdateUser(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserUseCase) SoftDeleteUser(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
