package route

import (
	"github.com/gin-gonic/gin"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/repository"
	"github.com/kirklin/boot-backend-go-clean/internal/usecase"
	"gorm.io/gorm"
)

func NewAuthRouter(db *gorm.DB, group *gin.RouterGroup) {
	ur := repository.NewUserRepository(db)
	ac := controller.NewAuthController(usecase.NewAuthUseCase(ur))

	group.POST("/register", ac.Register)
	group.POST("/login", ac.Login)
	group.POST("/refresh", ac.RefreshToken)
	group.POST("/logout", middleware.JWTAuthMiddleware(), ac.Logout)
}
