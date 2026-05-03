package route

import (
	"github.com/gin-gonic/gin"

	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/utils"
)

// registerUserRoutes registers user endpoints.
func (r *Router) registerUserRoutes(group *gin.RouterGroup, ctrl *controller.UserController) {
	users := group.Group("/users")
	users.Use(middleware.JWTAuthMiddleware(r.authenticator))
	{
		users.GET("/:id", ctrl.GetUser)

		// 获取当前用户信息
		users.GET("/current", ctrl.GetCurrentUser)

		// 更新用户资料，仅允许用户本人
		users.PUT("/:id", middleware.EnsureSelfMiddleware(utils.GetTargetUserIDFromParam), ctrl.UpdateUser)
	}
}
