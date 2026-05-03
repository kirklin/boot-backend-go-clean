package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	domainerrors "github.com/kirklin/boot-backend-go-clean/internal/domain/errors"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/middleware"
	testmock "github.com/kirklin/boot-backend-go-clean/internal/testutil/mock"
	"github.com/kirklin/boot-backend-go-clean/pkg/openapi"
)

func setupUserRouter(ctrl *UserController) *gin.Engine {
	spec := openapi.NewSpec("test", "0.0.0")
	r := gin.New()
	api := openapi.NewAPI(r.Group(""), spec)

	openapi.Get[GetUserInput, entity.User](api, "/users/:id", ctrl.GetUser,
		openapi.Message("User retrieved successfully"),
	)
	openapi.Put[UpdateUserInput, openapi.Empty](api, "/users/:id", ctrl.UpdateUser,
		openapi.Message("User updated successfully"),
	)
	openapi.Delete[DeleteUserInput, openapi.Empty](api, "/users/:id", ctrl.DeleteUser,
		openapi.Message("User deleted successfully"),
	)
	return r
}

func TestUserController_GetUser_Success(t *testing.T) {
	mockUC := new(testmock.MockUserUseCase)
	ctrl := NewUserController(mockUC)
	router := setupUserRouter(ctrl)

	mockUC.On("GetUserByID", mock.Anything, int64(1)).Return(
		&entity.User{ID: 1, Username: "kirk"}, nil,
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "kirk")
}

func TestUserController_GetUser_NotFound(t *testing.T) {
	mockUC := new(testmock.MockUserUseCase)
	ctrl := NewUserController(mockUC)
	router := setupUserRouter(ctrl)

	mockUC.On("GetUserByID", mock.Anything, int64(999)).Return(
		nil, domainerrors.ErrUserNotFound,
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/999", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "USER_NOT_FOUND")
}

func TestUserController_GetUser_InvalidID(t *testing.T) {
	mockUC := new(testmock.MockUserUseCase)
	ctrl := NewUserController(mockUC)
	router := setupUserRouter(ctrl)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/abc", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserController_DeleteUser_Success(t *testing.T) {
	mockUC := new(testmock.MockUserUseCase)
	ctrl := NewUserController(mockUC)
	router := setupUserRouter(ctrl)

	mockUC.On("SoftDeleteUser", mock.Anything, int64(1)).Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/users/1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "success", resp["status"])
}

func TestUserController_DeleteUser_NotFound(t *testing.T) {
	mockUC := new(testmock.MockUserUseCase)
	ctrl := NewUserController(mockUC)
	router := setupUserRouter(ctrl)

	mockUC.On("SoftDeleteUser", mock.Anything, int64(999)).Return(domainerrors.ErrUserNotFound)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/users/999", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── UpdateUser ───────────────────────────────────────────────────────────────

func TestUserController_UpdateUser_Success(t *testing.T) {
	mockUC := new(testmock.MockUserUseCase)
	ctrl := NewUserController(mockUC)
	router := setupUserRouter(ctrl)

	mockUC.On("UpdateUser", mock.Anything, mock.AnythingOfType("*entity.User")).Return(nil)

	body := toJSON(t, entity.User{ID: 1, Username: "kirk", Email: "kirk@example.com", Password: "securepass"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/users/1", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "User updated successfully")
}

func TestUserController_UpdateUser_ValidationFails(t *testing.T) {
	mockUC := new(testmock.MockUserUseCase)
	ctrl := NewUserController(mockUC)
	router := setupUserRouter(ctrl)

	mockUC.On("UpdateUser", mock.Anything, mock.AnythingOfType("*entity.User")).Return(
		domainerrors.ErrValidationFailed.WithMessage("username cannot be empty"),
	)

	body := toJSON(t, entity.User{ID: 1, Username: "", Email: "kirk@example.com", Password: "securepass"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/users/1", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserController_UpdateUser_InvalidJSON(t *testing.T) {
	mockUC := new(testmock.MockUserUseCase)
	ctrl := NewUserController(mockUC)
	router := setupUserRouter(ctrl)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/users/1", bytes.NewBufferString(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserController_DeleteUser_InvalidID(t *testing.T) {
	mockUC := new(testmock.MockUserUseCase)
	ctrl := NewUserController(mockUC)
	router := setupUserRouter(ctrl)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/users/abc", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── GetCurrentUser ───────────────────────────────────────────────────────────

func setupCurrentUserRouter(ctrl *UserController) *gin.Engine {
	spec := openapi.NewSpec("test", "0.0.0")
	r := gin.New()
	api := openapi.NewAPI(r.Group(""), spec)

	// Simulate JWT middleware that sets user ID in context
	fakeJWT := func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, int64(42))
		c.Next()
	}

	openapi.Get[openapi.Empty, entity.User](api, "/me", ctrl.GetCurrentUser,
		openapi.Middleware(fakeJWT),
		openapi.Message("User retrieved successfully"),
	)

	// For testing the no-auth case — MustUserID will panic, recover it.
	r.GET("/me-noauth", func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Unauthorized"})
			}
		}()
		ctx := openapi.TestContext(c)
		result, err := ctrl.GetCurrentUser(ctx, &openapi.Empty{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	return r
}

func TestUserController_GetCurrentUser_Success(t *testing.T) {
	mockUC := new(testmock.MockUserUseCase)
	ctrl := NewUserController(mockUC)
	router := setupCurrentUserRouter(ctrl)

	mockUC.On("GetUserByID", mock.Anything, int64(42)).Return(
		&entity.User{ID: 42, Username: "kirk"}, nil,
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/me", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "kirk")
}

func TestUserController_GetCurrentUser_NoAuth(t *testing.T) {
	mockUC := new(testmock.MockUserUseCase)
	ctrl := NewUserController(mockUC)
	router := setupCurrentUserRouter(ctrl)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/me-noauth", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
