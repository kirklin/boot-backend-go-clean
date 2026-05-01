package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity/response"

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
		ctx.JSON(http.StatusBadRequest, response.NewErrorResponse("Invalid input", err))
		return
	}

	resp, err := c.authUseCase.Register(ctx, &req)
	if err != nil {
		ctx.JSON(response.HTTPCodeFromError(err, http.StatusInternalServerError), response.NewErrorResponse("Registration failed", err))
		return
	}

	ctx.JSON(http.StatusCreated, response.NewSuccessResponse("User registered successfully", resp))
}

func (c *AuthController) Login(ctx *gin.Context) {
	var req entity.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewErrorResponse("Invalid input", err))
		return
	}

	resp, err := c.authUseCase.Login(ctx, &req)
	if err != nil {
		ctx.JSON(response.HTTPCodeFromError(err, http.StatusUnauthorized), response.NewErrorResponse("Login failed", err))
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccessResponse("Login successful", resp))
}

func (c *AuthController) RefreshToken(ctx *gin.Context) {
	var req entity.RefreshTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewErrorResponse("Invalid input", err))
		return
	}

	resp, err := c.authUseCase.RefreshToken(ctx, &req)
	if err != nil {
		ctx.JSON(response.HTTPCodeFromError(err, http.StatusUnauthorized), response.NewErrorResponse("Token refresh failed", err))
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccessResponse("Token refreshed successfully", resp))
}

func (c *AuthController) Logout(ctx *gin.Context) {
	var req entity.LogoutRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewErrorResponse("Invalid input", err))
		return
	}

	err := c.authUseCase.Logout(ctx, &req)
	if err != nil {
		ctx.JSON(response.HTTPCodeFromError(err, http.StatusInternalServerError), response.NewErrorResponse("Logout failed", err))
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccessResponse[any]("Logged out successfully", nil))
}
