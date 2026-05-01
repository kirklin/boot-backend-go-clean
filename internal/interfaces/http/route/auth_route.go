package route

import (
	"github.com/gin-gonic/gin"

	"github.com/kirklin/boot-backend-go-clean/pkg/configs"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/gateway"
	"github.com/kirklin/boot-backend-go-clean/internal/infrastructure/persistence"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	"github.com/kirklin/boot-backend-go-clean/internal/usecase"
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
)

func NewAuthRouter(db database.Database, group *gin.RouterGroup, config *configs.AppConfig, authenticator gateway.Authenticator) {
	ur := persistence.NewUserRepository(db)
	ac := controller.NewAuthController(usecase.NewAuthUseCase(ur, authenticator, config))

	authGroup := group.Group("/auth")
	authGroup.POST("/register", ac.Register)
	authGroup.POST("/login", ac.Login)
	authGroup.POST("/refresh", ac.RefreshToken)
	authGroup.POST("/logout", middleware.JWTAuthMiddleware(authenticator), ac.Logout)
}
