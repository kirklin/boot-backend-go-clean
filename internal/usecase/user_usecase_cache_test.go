package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	domainerrors "github.com/kirklin/boot-backend-go-clean/internal/domain/errors"
	testmock "github.com/kirklin/boot-backend-go-clean/internal/testutil/mock"
	"github.com/kirklin/boot-backend-go-clean/pkg/cache"
)

func newCachedUserUseCase(t *testing.T) (*testmock.MockUserRepository, cache.Cache, *userUseCase) {
	t.Helper()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	cacheClient := cache.NewRedisFromClient(client, cache.Config{NegativeTTL: time.Minute})
	repo := new(testmock.MockUserRepository)

	useCase, ok := NewUserUseCase(repo, cacheClient).(*userUseCase)
	require.True(t, ok)

	return repo, cacheClient, useCase
}

func TestUserUseCase_GetUserByID_SecondReadSkipsTheRepository(t *testing.T) {
	ctx := context.Background()
	repo, _, useCase := newCachedUserUseCase(t)

	stored := &entity.User{ID: 7, Username: "kirk", Email: "kirk@example.com"}
	repo.On("FindByID", mock.Anything, int64(7)).Return(stored, nil).Once()

	for range 3 {
		got, err := useCase.GetUserByID(ctx, 7)
		require.NoError(t, err)
		assert.Equal(t, "kirk", got.Username)
	}

	repo.AssertExpectations(t)
}

func TestUserUseCase_GetUserByID_NotFoundIsCachedOnce(t *testing.T) {
	ctx := context.Background()
	repo, _, useCase := newCachedUserUseCase(t)

	repo.On("FindByID", mock.Anything, int64(404)).Return(nil, domainerrors.ErrUserNotFound).Once()

	for range 3 {
		_, err := useCase.GetUserByID(ctx, 404)
		assert.ErrorIs(t, err, domainerrors.ErrUserNotFound,
			"callers see the domain error whether it came from the database or a tombstone")
	}

	repo.AssertExpectations(t)
}

func TestUserUseCase_GetUserByID_RepositoryErrorIsNotCached(t *testing.T) {
	ctx := context.Background()
	repo, _, useCase := newCachedUserUseCase(t)

	broken := assert.AnError
	repo.On("FindByID", mock.Anything, int64(7)).Return(nil, broken).Twice()

	for range 2 {
		_, err := useCase.GetUserByID(ctx, 7)
		assert.ErrorIs(t, err, broken)
	}

	repo.AssertExpectations(t)
}

func TestUserUseCase_UpdateUser_InvalidatesTheCachedUser(t *testing.T) {
	ctx := context.Background()
	repo, cacheClient, useCase := newCachedUserUseCase(t)

	original := &entity.User{ID: 7, Username: "kirk", Email: "kirk@example.com"}
	repo.On("FindByID", mock.Anything, int64(7)).Return(original, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil).Once()

	_, err := useCase.GetUserByID(ctx, 7)
	require.NoError(t, err)

	cached, err := cacheClient.Exists(ctx, "user:7")
	require.NoError(t, err)
	require.True(t, cached)

	updated := &entity.User{ID: 7, Username: "kirk-renamed", Email: "kirk@example.com", Password: "long-enough-password"}
	require.NoError(t, useCase.UpdateUser(ctx, updated))

	cached, err = cacheClient.Exists(ctx, "user:7")
	require.NoError(t, err)
	assert.False(t, cached, "the write must drop the entry so the next read rebuilds it")
}

func TestUserUseCase_SoftDeleteUser_InvalidatesTheCachedUser(t *testing.T) {
	ctx := context.Background()
	repo, cacheClient, useCase := newCachedUserUseCase(t)

	stored := &entity.User{ID: 7, Username: "kirk", Email: "kirk@example.com"}
	repo.On("FindByID", mock.Anything, int64(7)).Return(stored, nil)
	repo.On("SoftDelete", mock.Anything, int64(7)).Return(nil).Once()

	_, err := useCase.GetUserByID(ctx, 7)
	require.NoError(t, err)

	require.NoError(t, useCase.SoftDeleteUser(ctx, 7))

	cached, err := cacheClient.Exists(ctx, "user:7")
	require.NoError(t, err)
	assert.False(t, cached)
}

func TestUserUseCase_GetUsersByIDs(t *testing.T) {
	ctx := context.Background()
	repo, _, useCase := newCachedUserUseCase(t)

	stored := []*entity.User{
		{ID: 1, Username: "one"},
		{ID: 2, Username: "two"},
	}

	repo.On("FindByIDs", mock.Anything, mock.Anything).Return(stored, nil).Once()

	got, err := useCase.GetUsersByIDs(ctx, []int64{1, 2, 3})
	require.NoError(t, err)

	assert.Len(t, got, 2)
	assert.Equal(t, "one", got[1].Username)
	assert.Equal(t, "two", got[2].Username)
	assert.NotContains(t, got, int64(3), "an id with no row is absent, not a zero value")

	repo.AssertExpectations(t)
}

func TestUserUseCase_GetUsersByIDs_SecondReadSkipsTheRepository(t *testing.T) {
	ctx := context.Background()
	repo, _, useCase := newCachedUserUseCase(t)

	stored := []*entity.User{{ID: 1, Username: "one"}, {ID: 2, Username: "two"}}
	repo.On("FindByIDs", mock.Anything, mock.Anything).Return(stored, nil).Once()

	first, err := useCase.GetUsersByIDs(ctx, []int64{1, 2})
	require.NoError(t, err)
	require.Len(t, first, 2)

	second, err := useCase.GetUsersByIDs(ctx, []int64{1, 2})
	require.NoError(t, err)
	assert.Len(t, second, 2)

	repo.AssertExpectations(t)
}

func TestUserUseCase_GetUsersByIDs_RemembersAbsentIDs(t *testing.T) {
	ctx := context.Background()
	repo, _, useCase := newCachedUserUseCase(t)

	repo.On("FindByIDs", mock.Anything, []int64{404}).Return([]*entity.User{}, nil).Once()

	for range 3 {
		got, err := useCase.GetUsersByIDs(ctx, []int64{404})
		require.NoError(t, err)
		assert.Empty(t, got)
	}

	repo.AssertExpectations(t)
}

func TestUserUseCase_GetUsersByIDs_DeduplicatesIDs(t *testing.T) {
	ctx := context.Background()
	repo, _, useCase := newCachedUserUseCase(t)

	repo.On("FindByIDs", mock.Anything, []int64{1}).
		Return([]*entity.User{{ID: 1, Username: "one"}}, nil).Once()

	got, err := useCase.GetUsersByIDs(ctx, []int64{1, 1, 1})
	require.NoError(t, err)
	assert.Len(t, got, 1)

	repo.AssertExpectations(t)
}

func TestUserUseCase_GetUsersByIDs_Empty(t *testing.T) {
	repo, _, useCase := newCachedUserUseCase(t)

	got, err := useCase.GetUsersByIDs(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, got)

	repo.AssertNotCalled(t, "FindByIDs", mock.Anything, mock.Anything)
}
