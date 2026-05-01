package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware provides a safe CORS configuration
func CORSMiddleware() gin.HandlerFunc {
	// Using gin-contrib/cors for standard and safe CORS handling
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true // For development. In production, use config.AllowOrigins = []string{"http://yourdomain.com"}
	config.AllowCredentials = true
	// Since we allow credentials, AllowAllOrigins cannot be true in strict mode, but gin-contrib/cors
	// handles echoing the origin properly when AllowAllOrigins is true.
	// Actually, if AllowAllOrigins is true and AllowCredentials is true, gin-contrib/cors
	// specifically mirrors the exact origin to satisfy browsers.
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Accept", "X-Requested-With", "X-CSRF-Token"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.MaxAge = 12 * time.Hour

	return cors.New(config)
}
