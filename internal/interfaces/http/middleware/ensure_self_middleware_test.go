package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── Helper ───────────────────────────────────────────────────────────────────

func setupEnsureSelfRouter(getTargetUserID func(c *gin.Context) (int64, error)) *gin.Engine {
	r := gin.New()
	// Simulate JWT middleware setting user ID
	r.Use(func(c *gin.Context) {
		if uid := c.GetHeader("X-Test-UserID"); uid != "" {
			id, _ := strconv.ParseInt(uid, 10, 64)
			c.Set("x-user-id", id)
		}
		c.Next()
	})
	r.Use(EnsureSelfMiddleware(getTargetUserID))
	r.GET("/users/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestEnsureSelf_MatchingUser(t *testing.T) {
	getTarget := func(c *gin.Context) (int64, error) {
		return strconv.ParseInt(c.Param("id"), 10, 64)
	}
	router := setupEnsureSelfRouter(getTarget)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/42", nil)
	req.Header.Set("X-Test-UserID", "42")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEnsureSelf_DifferentUser(t *testing.T) {
	getTarget := func(c *gin.Context) (int64, error) {
		return strconv.ParseInt(c.Param("id"), 10, 64)
	}
	router := setupEnsureSelfRouter(getTarget)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/99", nil)
	req.Header.Set("X-Test-UserID", "42")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "PERMISSION_DENIED")
}

func TestEnsureSelf_NoAuthenticatedUser(t *testing.T) {
	getTarget := func(c *gin.Context) (int64, error) {
		return 1, nil
	}
	router := setupEnsureSelfRouter(getTarget)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/1", nil)
	// No X-Test-UserID header → no user in context
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Unable to identify authenticated user")
}

func TestEnsureSelf_InvalidTargetUserID(t *testing.T) {
	getTarget := func(c *gin.Context) (int64, error) {
		return 0, fmt.Errorf("invalid id format")
	}
	router := setupEnsureSelfRouter(getTarget)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/abc", nil)
	req.Header.Set("X-Test-UserID", "42")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid target user ID")
}
