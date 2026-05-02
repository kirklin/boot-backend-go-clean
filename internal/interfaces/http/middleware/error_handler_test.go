package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	domainerrors "github.com/kirklin/boot-backend-go-clean/internal/domain/errors"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func TestErrorHandler_WithAppError(t *testing.T) {
	r := gin.New()
	r.Use(ErrorHandler())
	r.GET("/test", func(c *gin.Context) {
		_ = c.Error(domainerrors.ErrUserNotFound)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	errObj := resp["error"].(map[string]any)
	assert.Equal(t, "USER_NOT_FOUND", errObj["code"])
}

func TestErrorHandler_WithGenericError(t *testing.T) {
	r := gin.New()
	r.Use(ErrorHandler())
	r.GET("/test", func(c *gin.Context) {
		_ = c.Error(errors.New("unexpected panic"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestErrorHandler_NoErrors(t *testing.T) {
	r := gin.New()
	r.Use(ErrorHandler())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestErrorHandler_ResponseAlreadyWritten(t *testing.T) {
	r := gin.New()
	r.Use(ErrorHandler())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "already written"})
		_ = c.Error(domainerrors.ErrInternal)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	// Should keep the original 400, not overwrite with 500
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
