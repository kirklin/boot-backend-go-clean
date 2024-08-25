package controller

import (
	"github.com/gin-gonic/gin"
	"net/http"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/usecase"
)

type AuthController struct {
	authUseCase usecase.AuthUseCase
}

func NewAuthController(authUseCase usecase.AuthUseCase) *AuthController {
	return &AuthController{
		authUseCase: authUseCase,
	}
}

func (c *AuthController) Register(ctx *gin.Context) {
	var req entity.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, entity.ErrorResponse{
			Status:  "error",
			Message: "Invalid input",
			Error:   err.Error(),
		})
		return
	}

	resp, err := c.authUseCase.Register(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, entity.ErrorResponse{
			Status:  "error",
			Message: "Registration failed",
			Error:   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, entity.SuccessResponse{
		Status:  "success",
		Message: "User registered successfully",
		Data:    resp,
	})
}

func (c *AuthController) Login(ctx *gin.Context) {
	var req entity.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, entity.ErrorResponse{
			Status:  "error",
			Message: "Invalid input",
			Error:   err.Error(),
		})
		return
	}

	resp, err := c.authUseCase.Login(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, entity.ErrorResponse{
			Status:  "error",
			Message: "Login failed",
			Error:   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, entity.SuccessResponse{
		Status:  "success",
		Message: "Login successful",
		Data:    resp,
	})
}

func (c *AuthController) RefreshToken(ctx *gin.Context) {
	var req entity.RefreshTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, entity.ErrorResponse{
			Status:  "error",
			Message: "Invalid input",
			Error:   err.Error(),
		})
		return
	}

	resp, err := c.authUseCase.RefreshToken(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, entity.ErrorResponse{
			Status:  "error",
			Message: "Token refresh failed",
			Error:   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, entity.SuccessResponse{
		Status:  "success",
		Message: "Token refreshed successfully",
		Data:    resp,
	})
}

func (c *AuthController) Logout(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, entity.ErrorResponse{
			Status:  "error",
			Message: "User not authenticated",
		})
		return
	}

	err := c.authUseCase.Logout(ctx, userID.(uint))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, entity.ErrorResponse{
			Status:  "error",
			Message: "Logout failed",
			Error:   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, entity.SuccessResponse{
		Status:  "success",
		Message: "Logged out successfully",
	})
}
