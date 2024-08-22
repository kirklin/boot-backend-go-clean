package route

import (
	"github.com/kirklin/boot-backend-go-clean/pkg/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures the routes for the application
func SetupRoutes(router *gin.Engine, db database.Database) {
	// Health check route
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// Add more routes here
	// Example:
	// router.GET("/api/users", controllers.GetUsers)
	// router.POST("/api/users", controllers.CreateUser)
}
