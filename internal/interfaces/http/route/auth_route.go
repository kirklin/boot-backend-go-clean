package route

import (
	"github.com/gin-gonic/gin"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/gateway"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
)

// NewAuthRouter registers auth endpoints.
// All dependencies are pre-built and injected from the Composition Root (app.Initialize).
func NewAuthRouter(group *gin.RouterGroup, ac *controller.AuthController, authenticator gateway.Authenticator) {
	authGroup := group.Group("/auth")
	authGroup.POST("/register", ac.Register)
	authGroup.POST("/login", ac.Login)
	authGroup.POST("/refresh", ac.RefreshToken)
	authGroup.POST("/logout", middleware.JWTAuthMiddleware(authenticator), ac.Logout)
}
