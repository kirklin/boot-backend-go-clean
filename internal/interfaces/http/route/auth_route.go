package route

import (
	"github.com/gin-gonic/gin"

	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/controller"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
)

// registerAuthRoutes registers auth endpoints.
func (r *Router) registerAuthRoutes(group *gin.RouterGroup, ctrl *controller.AuthController) {
	auth := group.Group("/auth")
	auth.POST("/register", ctrl.Register)
	auth.POST("/login", ctrl.Login)
	auth.POST("/refresh", ctrl.RefreshToken)
	auth.POST("/logout", middleware.JWTAuthMiddleware(r.authenticator), ctrl.Logout)
}
