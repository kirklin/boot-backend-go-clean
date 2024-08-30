package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity/response"
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
		resp := response.NewErrorResponse("Invalid input", err)
		ctx.JSON(http.StatusBadRequest, resp)
		return
	}

	resp, err := c.authUseCase.Register(ctx, &req)
	if err != nil {
		errorResp := response.NewErrorResponse("Registration failed", err)
		ctx.JSON(http.StatusInternalServerError, errorResp)
		return
	}

	successResp := response.NewSuccessResponse("User registered successfully", resp)
	ctx.JSON(http.StatusCreated, successResp)
}

func (c *AuthController) Login(ctx *gin.Context) {
	var req entity.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp := response.NewErrorResponse("Invalid input", err)
		ctx.JSON(http.StatusBadRequest, resp)
		return
	}

	resp, err := c.authUseCase.Login(ctx, &req)
	if err != nil {
		errorResp := response.NewErrorResponse("Login failed", err)
		ctx.JSON(http.StatusUnauthorized, errorResp)
		return
	}

	successResp := response.NewSuccessResponse("Login successful", resp)
	ctx.JSON(http.StatusOK, successResp)
}

func (c *AuthController) RefreshToken(ctx *gin.Context) {
	var req entity.RefreshTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp := response.NewErrorResponse("Invalid input", err)
		ctx.JSON(http.StatusBadRequest, resp)
		return
	}

	resp, err := c.authUseCase.RefreshToken(ctx, &req)
	if err != nil {
		errorResp := response.NewErrorResponse("Token refresh failed", err)
		ctx.JSON(http.StatusUnauthorized, errorResp)
		return
	}

	successResp := response.NewSuccessResponse("Token refreshed successfully", resp)
	ctx.JSON(http.StatusOK, successResp)
}

func (c *AuthController) Logout(ctx *gin.Context) {
	var req entity.LogoutRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		resp := response.NewErrorResponse("Invalid input", err)
		ctx.JSON(http.StatusBadRequest, resp)
		return
	}

	err := c.authUseCase.Logout(ctx, &req)
	if err != nil {
		errorResp := response.NewErrorResponse("Logout failed", err)
		ctx.JSON(http.StatusInternalServerError, errorResp)
		return
	}

	successResp := response.NewSuccessResponse[any]("Logged out successfully", nil)
	ctx.JSON(http.StatusOK, successResp)
}
