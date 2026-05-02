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
)

func setupUserRouter(ctrl *UserController) *gin.Engine {
	r := gin.New()
	r.GET("/users/:id", ctrl.GetUser)
	r.PUT("/users/:id", ctrl.UpdateUser)
	r.DELETE("/users/:id", ctrl.DeleteUser)
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
	r := gin.New()
	r.GET("/me", func(c *gin.Context) {
		// Simulate JWT middleware setting user ID
		c.Set(middleware.ContextKeyUserID, int64(42))
		c.Next()
	}, ctrl.GetCurrentUser)
	r.GET("/me-noauth", ctrl.GetCurrentUser) // Without user context
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
