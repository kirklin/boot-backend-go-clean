package route

import (
	"github.com/gin-gonic/gin"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/gateway"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/utils"
)

// NewUserRouter registers user endpoints.
// All dependencies are pre-built and injected from the Composition Root (app.Initialize).
func NewUserRouter(group *gin.RouterGroup, uc *controller.UserController, authenticator gateway.Authenticator) {
	userRoutes := group.Group("/users")
	userRoutes.Use(middleware.JWTAuthMiddleware(authenticator))
	{
		userRoutes.GET("/:id", uc.GetUser)

		// 获取当前用户信息
		userRoutes.GET("/current", uc.GetCurrentUser)

		// 更新用户资料，仅允许用户本人
		userRoutes.PUT("/:id", middleware.EnsureSelfMiddleware(utils.GetTargetUserIDFromParam), uc.UpdateUser)
	}
}
