package usecase

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	domainerrors "github.com/kirklin/boot-backend-go-clean/internal/domain/errors"
	testmock "github.com/kirklin/boot-backend-go-clean/internal/testutil/mock"
)

// ─── GetUserByID ──────────────────────────────────────────────────────────────

func TestUserUseCase_GetUserByID_Success(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	uc := NewUserUseCase(repo)

	expected := &entity.User{ID: 1, Username: "kirk", Email: "kirk@example.com"}
	repo.On("FindByID", mock.Anything, int64(1)).Return(expected, nil)

	user, err := uc.GetUserByID(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, expected, user)
	repo.AssertExpectations(t)
}

func TestUserUseCase_GetUserByID_NotFound(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	uc := NewUserUseCase(repo)

	repo.On("FindByID", mock.Anything, int64(999)).Return(nil, domainerrors.ErrUserNotFound)

	user, err := uc.GetUserByID(context.Background(), 999)

	assert.ErrorIs(t, err, domainerrors.ErrUserNotFound)
	assert.Nil(t, user)
}

// ─── UpdateUser ───────────────────────────────────────────────────────────────

func TestUserUseCase_UpdateUser_Success(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	uc := NewUserUseCase(repo)

	user := &entity.User{ID: 1, Username: "kirk", Email: "kirk@example.com", Password: "securepass"}
	repo.On("FindByID", mock.Anything, int64(1)).Return(user, nil)
	repo.On("Update", mock.Anything, user).Return(nil)

	err := uc.UpdateUser(context.Background(), user)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUserUseCase_UpdateUser_ValidationFails(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	uc := NewUserUseCase(repo)

	user := &entity.User{ID: 1, Username: "", Email: "kirk@example.com", Password: "securepass"}

	err := uc.UpdateUser(context.Background(), user)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username cannot be empty")
	// FindByID should NOT have been called because validation failed first
	repo.AssertNotCalled(t, "FindByID")
}

func TestUserUseCase_UpdateUser_NotFound(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	uc := NewUserUseCase(repo)

	user := &entity.User{ID: 999, Username: "kirk", Email: "kirk@example.com", Password: "securepass"}
	repo.On("FindByID", mock.Anything, int64(999)).Return(nil, domainerrors.ErrUserNotFound)

	err := uc.UpdateUser(context.Background(), user)

	assert.ErrorIs(t, err, domainerrors.ErrUserNotFound)
	repo.AssertNotCalled(t, "Update")
}

// ─── SoftDeleteUser ───────────────────────────────────────────────────────────

func TestUserUseCase_SoftDeleteUser_Success(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	uc := NewUserUseCase(repo)

	user := &entity.User{ID: 1, Username: "kirk"}
	repo.On("FindByID", mock.Anything, int64(1)).Return(user, nil)
	repo.On("SoftDelete", mock.Anything, int64(1)).Return(nil)

	err := uc.SoftDeleteUser(context.Background(), 1)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUserUseCase_SoftDeleteUser_NotFound(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	uc := NewUserUseCase(repo)

	repo.On("FindByID", mock.Anything, int64(999)).Return(nil, domainerrors.ErrUserNotFound)

	err := uc.SoftDeleteUser(context.Background(), 999)

	assert.ErrorIs(t, err, domainerrors.ErrUserNotFound)
	repo.AssertNotCalled(t, "SoftDelete")
}

// ─── Error Branches ───────────────────────────────────────────────────────────

func TestUserUseCase_UpdateUser_UpdateFails(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	uc := NewUserUseCase(repo)

	user := &entity.User{ID: 1, Username: "kirk", Email: "kirk@example.com", Password: "securepass"}
	repo.On("FindByID", mock.Anything, int64(1)).Return(user, nil)
	repo.On("Update", mock.Anything, user).Return(fmt.Errorf("db write error"))

	err := uc.UpdateUser(context.Background(), user)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db write error")
}

func TestUserUseCase_SoftDeleteUser_DeleteFails(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	uc := NewUserUseCase(repo)

	user := &entity.User{ID: 1, Username: "kirk"}
	repo.On("FindByID", mock.Anything, int64(1)).Return(user, nil)
	repo.On("SoftDelete", mock.Anything, int64(1)).Return(fmt.Errorf("db write error"))

	err := uc.SoftDeleteUser(context.Background(), 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db write error")
}
