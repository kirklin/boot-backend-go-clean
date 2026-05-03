package route

import (
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/utils"
	"github.com/kirklin/boot-backend-go-clean/pkg/openapi"
)

// registerUserRoutes registers user endpoints with typed handlers.
func (r *Router) registerUserRoutes(api *openapi.API, ctrl *controller.UserController) {
	users := api.Group("/users", middleware.JWTAuthMiddleware(r.authenticator))

	openapi.Get[openapi.Empty, entity.User](
		users, "/current", ctrl.GetCurrentUser,
		openapi.Summary("获取当前登录用户信息"),
		openapi.Tags("User"),
		openapi.Security("bearer"),
		openapi.Message("User retrieved successfully"),
	)

	openapi.Get[controller.GetUserInput, entity.User](
		users, "/:id", ctrl.GetUser,
		openapi.Summary("获取用户信息"),
		openapi.Tags("User"),
		openapi.Security("bearer"),
		openapi.Message("User retrieved successfully"),
	)

	openapi.Put[controller.UpdateUserInput, openapi.Empty](
		users, "/:id", ctrl.UpdateUser,
		openapi.Summary("更新用户信息（仅本人）"),
		openapi.Tags("User"),
		openapi.Security("bearer"),
		openapi.Middleware(middleware.EnsureSelfMiddleware(utils.GetTargetUserIDFromParam)),
		openapi.Message("User updated successfully"),
	)

	openapi.Delete[controller.DeleteUserInput, openapi.Empty](
		users, "/:id", ctrl.DeleteUser,
		openapi.Summary("删除用户"),
		openapi.Tags("User"),
		openapi.Security("bearer"),
		openapi.Message("User deleted successfully"),
	)
}
