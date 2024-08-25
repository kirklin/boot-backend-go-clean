package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
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
	userID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, entity.ErrorResponse{
			Status:  "error",
			Message: "Invalid user ID",
			Error:   err.Error(),
		})
		return
	}

	user, err := c.userUseCase.GetUserByID(ctx, uint(userID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, entity.ErrorResponse{
			Status:  "error",
			Message: "Failed to get user",
			Error:   err.Error(),
		})
		return
	}

	if user == nil {
		ctx.JSON(http.StatusNotFound, entity.ErrorResponse{
			Status:  "error",
			Message: "User not found",
		})
		return
	}

	ctx.JSON(http.StatusOK, entity.SuccessResponse{
		Status:  "success",
		Message: "User retrieved successfully",
		Data:    user,
	})
}

func (c *UserController) UpdateUser(ctx *gin.Context) {
	var user entity.User
	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, entity.ErrorResponse{
			Status:  "error",
			Message: "Invalid input",
			Error:   err.Error(),
		})
		return
	}

	err := c.userUseCase.UpdateUser(ctx, &user)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, entity.ErrorResponse{
			Status:  "error",
			Message: "Failed to update user",
			Error:   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, entity.SuccessResponse{
		Status:  "success",
		Message: "User updated successfully",
		Data:    user,
	})
}

func (c *UserController) DeleteUser(ctx *gin.Context) {
	userID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, entity.ErrorResponse{
			Status:  "error",
			Message: "Invalid user ID",
			Error:   err.Error(),
		})
		return
	}

	err = c.userUseCase.SoftDeleteUser(ctx, uint(userID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, entity.ErrorResponse{
			Status:  "error",
			Message: "Failed to delete user",
			Error:   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, entity.SuccessResponse{
		Status:  "success",
		Message: "User deleted successfully",
	})
}
