package route

import (
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/utils"
	"github.com/kirklin/boot-backend-go-clean/pkg/openapi"
)

// registerUserRoutes registers user endpoints.
func (r *Router) registerUserRoutes(api *openapi.API, ctrl *controller.UserController) {
	users := api.Group("/users", middleware.JWTAuthMiddleware(r.authenticator))

	openapi.Get[openapi.Empty, entity.User](
		users, "/:id", ctrl.GetUser,
		openapi.Summary("获取用户信息"),
		openapi.Tags("User"),
		openapi.Security("bearer"),
	)

	// 获取当前用户信息
	openapi.Get[openapi.Empty, entity.User](
		users, "/current", ctrl.GetCurrentUser,
		openapi.Summary("获取当前登录用户信息"),
		openapi.Tags("User"),
		openapi.Security("bearer"),
	)

	// 更新用户资料，仅允许用户本人
	openapi.Put[entity.User, openapi.Empty](
		users, "/:id", ctrl.UpdateUser,
		openapi.Summary("更新用户信息（仅本人）"),
		openapi.Tags("User"),
		openapi.Security("bearer"),
		openapi.Middleware(middleware.EnsureSelfMiddleware(utils.GetTargetUserIDFromParam)),
	)
}
