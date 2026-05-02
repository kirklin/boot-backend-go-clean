package controller

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity/response"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/usecase"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/humaerr"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
)

type UserController struct {
	userUseCase usecase.UserUseCase
}

func NewUserController(userUseCase usecase.UserUseCase) *UserController {
	return &UserController{
		userUseCase: userUseCase,
	}
}

// --- Huma Input/Output types ---

type GetUserInput struct {
	ID int64 `path:"id" doc:"User ID"`
}

type GetUserOutput struct {
	Body response.Response[*entity.User]
}

type GetCurrentUserOutput struct {
	Body response.Response[*entity.User]
}

// UpdateUserBody is the request body for updating a user.
// ID comes from the path parameter, not the body.
type UpdateUserBody struct {
	Username  string  `json:"username" doc:"Username"`
	Email     string  `json:"email" doc:"Email address"`
	AvatarURL *string `json:"avatar_url,omitempty" doc:"Avatar URL"`
}

type UpdateUserInput struct {
	ID   int64          `path:"id" doc:"User ID"`
	Body UpdateUserBody `json:"body"`
}

type UpdateUserOutput struct {
	Body response.Response[any]
}

type DeleteUserInput struct {
	ID int64 `path:"id" doc:"User ID"`
}

type DeleteUserOutput struct {
	Body response.Response[any]
}

// --- Huma handlers ---

func (c *UserController) GetUser(ctx context.Context, input *GetUserInput) (*GetUserOutput, error) {
	user, err := c.userUseCase.GetUserByID(ctx, input.ID)
	if err != nil {
		return nil, humaerr.NewHumaError(http.StatusInternalServerError, "Failed to get user", err)
	}

	return &GetUserOutput{
		Body: response.NewSuccessResponse("User retrieved successfully", user),
	}, nil
}

// GetCurrentUser retrieves the current authenticated user.
// It extracts the user ID from the gin.Context that was set by JWTAuthMiddleware.
func (c *UserController) GetCurrentUser(ctx context.Context, input *struct{}) (*GetCurrentUserOutput, error) {
	// We need to get the gin.Context to access the user ID set by JWTAuthMiddleware.
	// This is done via huma's context chain: context.Context -> huma.Context -> gin.Context.
	ginCtxVal := ctx.Value(HumaContextKey)
	if ginCtxVal == nil {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	ginCtx, ok := ginCtxVal.(*gin.Context)
	if !ok {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	userID, exists := middleware.GetUserIDFromContext(ginCtx)
	if !exists {
		return nil, huma.Error401Unauthorized("Unauthorized")
	}

	user, err := c.userUseCase.GetUserByID(ctx, userID)
	if err != nil {
		return nil, humaerr.NewHumaError(http.StatusInternalServerError, "Failed to get user", err)
	}

	return &GetCurrentUserOutput{
		Body: response.NewSuccessResponse("User retrieved successfully", user),
	}, nil
}

func (c *UserController) UpdateUser(ctx context.Context, input *UpdateUserInput) (*UpdateUserOutput, error) {
	user := &entity.User{
		ID:        input.ID,
		Username:  input.Body.Username,
		Email:     input.Body.Email,
		AvatarURL: input.Body.AvatarURL,
	}
	if err := c.userUseCase.UpdateUser(ctx, user); err != nil {
		return nil, humaerr.NewHumaError(http.StatusInternalServerError, "Failed to update user", err)
	}

	return &UpdateUserOutput{
		Body: response.NewSuccessResponse[any]("User updated successfully", nil),
	}, nil
}

func (c *UserController) DeleteUser(ctx context.Context, input *DeleteUserInput) (*DeleteUserOutput, error) {
	if err := c.userUseCase.SoftDeleteUser(ctx, input.ID); err != nil {
		return nil, humaerr.NewHumaError(http.StatusInternalServerError, "Failed to delete user", err)
	}

	return &DeleteUserOutput{
		Body: response.NewSuccessResponse[any]("User deleted successfully", nil),
	}, nil
}

// humaContextKey is used to pass huma.Context through context.Context
// so that handlers can access the underlying gin.Context when needed
// (e.g. to read values set by gin middlewares like JWT userID).
// ContextKeyType is used for context key typing to avoid collisions.
type ContextKeyType string

// HumaContextKey is used to pass gin.Context through context.Context
// so that huma handlers can access values set by gin middlewares.
const HumaContextKey ContextKeyType = "gin.Context"
