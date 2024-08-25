package route

import (
	"github.com/gin-gonic/gin"

	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/persistence"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/internal/usecase"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
)

func NewAuthRouter(db database.Database, group *gin.RouterGroup) {
	ur := persistence.NewUserRepository(db)
	ac := controller.NewAuthController(usecase.NewAuthUseCase(ur))

	group.POST("/register", ac.Register)
	group.POST("/login", ac.Login)
	group.POST("/refresh", ac.RefreshToken)
	group.POST("/logout", middleware.JWTAuthMiddleware(), ac.Logout)
}
