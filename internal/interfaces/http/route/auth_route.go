package route

import (
	"net/http"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/pkg/openapi"
)

// registerAuthRoutes registers auth endpoints with typed handlers.
func (r *Router) registerAuthRoutes(api *openapi.API, ctrl *controller.AuthController) {
	auth := api.Group("/auth")

	openapi.Post[controller.RegisterInput, entity.RegisterResponse](
		auth, "/register", ctrl.Register,
		openapi.Summary("注册新用户"),
		openapi.Tags("Auth"),
		openapi.Status(http.StatusCreated),
		openapi.Message("User registered successfully"),
	)

	openapi.Post[controller.LoginInput, entity.LoginResponse](
		auth, "/login", ctrl.Login,
		openapi.Summary("用户登录"),
		openapi.Tags("Auth"),
		openapi.Message("Login successful"),
	)

	openapi.Post[controller.RefreshInput, entity.RefreshTokenResponse](
		auth, "/refresh", ctrl.RefreshToken,
		openapi.Summary("刷新访问令牌"),
		openapi.Tags("Auth"),
		openapi.Message("Token refreshed successfully"),
	)

	openapi.Post[controller.LogoutInput, openapi.Empty](
		auth, "/logout", ctrl.Logout,
		openapi.Summary("用户登出"),
		openapi.Tags("Auth"),
		openapi.Security("bearer"),
		openapi.Middleware(middleware.JWTAuthMiddleware(r.authenticator)),
		openapi.Message("Logged out successfully"),
	)
}
