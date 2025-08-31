package route

import (
	"github.com/gin-gonic/gin"
	"github.com/kirklin/boot-backend-go-clean/pkg/configs"

	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/persistence"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/internal/usecase"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
)

func NewUserRouter(db database.Database, group *gin.RouterGroup, config *configs.AppConfig) {
	ur := persistence.NewUserRepository(db)
	uc := controller.NewUserController(usecase.NewUserUseCase(ur))

	userRoutes := group.Group("/users")
	userRoutes.Use(middleware.JWTAuthMiddleware())
	{
		userRoutes.GET("/:id", uc.GetUser)

		// 获取当前用户信息
		userRoutes.GET("/current", uc.GetCurrentUser)

		// 更新用户资料，仅允许用户本人
		userRoutes.PUT("/:id", middleware.EnsureSelfMiddleware(middleware.GetTargetUserIDFromParam), uc.UpdateUser)
	}
}
