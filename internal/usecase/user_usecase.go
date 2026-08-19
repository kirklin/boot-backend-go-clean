package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	domainerrors "github.com/kirklin/boot-backend-go-clean/internal/domain/errors"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/repository"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/usecase"
	"github.com/kirklin/boot-backend-go-clean/pkg/cache"
)

const userCacheTTL = 10 * time.Minute

const userStaleGrace = 5 * time.Minute

func userCacheKey(id int64) string {
	return fmt.Sprintf("user:%d", id)
}

type userUseCase struct {
	userRepo repository.UserRepository
	cache    cache.Cache
}

// NewUserUseCase builds the user use case; cacheClient is never nil.
func NewUserUseCase(userRepo repository.UserRepository, cacheClient cache.Cache) usecase.UserUseCase {
	return &userUseCase{
		userRepo: userRepo,
		cache:    cacheClient,
	}
}

func (u *userUseCase) GetUserByID(ctx context.Context, id int64) (*entity.User, error) {
	user, err := cache.GetOrLoad(ctx, u.cache, userCacheKey(id), userCacheTTL,
		func(ctx context.Context) (*entity.User, error) {
			found, err := u.userRepo.FindByID(ctx, id)
			if err != nil {
				if errors.Is(err, domainerrors.ErrUserNotFound) {
					return nil, cache.ErrNotFound
				}
				return nil, err
			}
			return found, nil
		},
		cache.WithStaleWhileRevalidate(userStaleGrace),
	)
	if errors.Is(err, cache.ErrNotFound) {
		return nil, domainerrors.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (u *userUseCase) GetUsersByIDs(ctx context.Context, ids []int64) (map[int64]*entity.User, error) {
	keys := make([]string, 0, len(ids))
	idByKey := make(map[string]int64, len(ids))
	for _, id := range ids {
		key := userCacheKey(id)
		if _, seen := idByKey[key]; seen {
			continue
		}
		idByKey[key] = id
		keys = append(keys, key)
	}

	byKey, err := cache.GetOrLoadBatch(ctx, u.cache, keys, userCacheTTL,
		func(ctx context.Context, missing []string) (map[string]*entity.User, error) {
			missingIDs := make([]int64, 0, len(missing))
			for _, key := range missing {
				missingIDs = append(missingIDs, idByKey[key])
			}

			users, err := u.userRepo.FindByIDs(ctx, missingIDs)
			if err != nil {
				return nil, err
			}

			loaded := make(map[string]*entity.User, len(users))
			for _, user := range users {
				loaded[userCacheKey(user.ID)] = user
			}
			return loaded, nil
		})
	if err != nil {
		return nil, err
	}

	users := make(map[int64]*entity.User, len(byKey))
	for key, user := range byKey {
		users[idByKey[key]] = user
	}
	return users, nil
}

func (u *userUseCase) UpdateUser(ctx context.Context, user *entity.User) error {
	if err := user.Validate(); err != nil {
		return err
	}

	if _, err := u.userRepo.FindByID(ctx, user.ID); err != nil {
		return err
	}

	if err := u.userRepo.Update(ctx, user); err != nil {
		return err
	}

	u.invalidate(ctx, user.ID)
	return nil
}

func (u *userUseCase) SoftDeleteUser(ctx context.Context, id int64) error {
	if _, err := u.userRepo.FindByID(ctx, id); err != nil {
		return err
	}

	if err := u.userRepo.SoftDelete(ctx, id); err != nil {
		return err
	}

	u.invalidate(ctx, id)
	return nil
}

func (u *userUseCase) invalidate(ctx context.Context, id int64) {
	_ = u.cache.Delete(ctx, userCacheKey(id))
}
