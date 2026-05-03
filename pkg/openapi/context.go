package openapi

import (
	"context"

	"github.com/gin-gonic/gin"
)

type contextKey int

const ginContextKey contextKey = iota

// withGinContext stores the gin.Context in a standard context.Context,
// allowing typed handlers to access Gin-specific data (JWT claims, etc.)
// when needed.
func withGinContext(c *gin.Context) context.Context {
	return context.WithValue(c.Request.Context(), ginContextKey, c)
}

// GinContext retrieves the underlying gin.Context from a context.Context.
// This is the escape hatch for handlers that need Gin-specific features
// (e.g. setting response headers, accessing middleware values).
func GinContext(ctx context.Context) *gin.Context {
	gc, _ := ctx.Value(ginContextKey).(*gin.Context)
	return gc
}

// MustUserID extracts the authenticated user ID from context.
// The value is set by JWTAuthMiddleware. Panics if not present.
func MustUserID(ctx context.Context) int64 {
	gc := GinContext(ctx)
	if gc == nil {
		panic("openapi: MustUserID called without gin context")
	}
	val, exists := gc.Get("x-user-id")
	if !exists {
		panic("openapi: MustUserID called without authenticated user")
	}
	return val.(int64)
}

// UserID extracts the authenticated user ID from context.
// Returns (0, false) if not authenticated.
func UserID(ctx context.Context) (int64, bool) {
	gc := GinContext(ctx)
	if gc == nil {
		return 0, false
	}
	val, exists := gc.Get("x-user-id")
	if !exists {
		return 0, false
	}
	id, ok := val.(int64)
	return id, ok
}

// TestContext creates a context.Context from a gin.Context for testing.
// This allows test code to call typed handlers directly while still
// having access to gin.Context values (e.g. middleware-set user IDs).
func TestContext(c *gin.Context) context.Context {
	return withGinContext(c)
}
