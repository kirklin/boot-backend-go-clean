package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity/response"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/usecase"
)

type UserController struct {
	userUseCase usecase.UserUseCase
}

func NewUserController(userUseCase usecase.UserUseCase) *UserController {
	return &UserController{
		userUseCase: userUseCase,
	}
}

func (c *UserController) GetUser(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewErrorResponse("Invalid user ID", err))
		return
	}

	user, err := c.userUseCase.GetUserByID(ctx, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewErrorResponse("Failed to get user", err))
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccessResponse("User retrieved successfully", user))
}

func (c *UserController) UpdateUser(ctx *gin.Context) {
	var user entity.User
	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewErrorResponse("Invalid input", err))
		return
	}

	if err := c.userUseCase.UpdateUser(ctx, &user); err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewErrorResponse("Failed to update user", err))
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccessResponse[any]("User updated successfully", nil))
}

func (c *UserController) DeleteUser(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewErrorResponse("Invalid user ID", err))
		return
	}

	if err := c.userUseCase.SoftDeleteUser(ctx, userID); err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewErrorResponse("Failed to delete user", err))
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccessResponse[any]("User deleted successfully", nil))
}
