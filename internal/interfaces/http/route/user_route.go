package route

import (
	"github.com/gin-gonic/gin"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/repository"
	"github.com/kirklin/boot-backend-go-clean/internal/usecase"
	"gorm.io/gorm"
)

func NewUserRouter(db *gorm.DB, group *gin.RouterGroup) {
	ur := repository.NewUserRepository(db)
	uc := controller.NewUserController(usecase.NewUserUseCase(ur))

	userRoutes := group.Group("/users")
	userRoutes.Use(middleware.JWTAuthMiddleware())
	{
		userRoutes.GET("/:id", uc.GetUser)
		userRoutes.PUT("/:id", uc.UpdateUser)
		userRoutes.DELETE("/:id", uc.DeleteUser)
	}
}
