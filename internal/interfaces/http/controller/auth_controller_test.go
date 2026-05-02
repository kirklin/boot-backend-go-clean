package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	domainerrors "github.com/kirklin/boot-backend-go-clean/internal/domain/errors"
	"github.com/kirklin/boot-backend-go-clean/internal/interfaces/http/humaerr"
	testmock "github.com/kirklin/boot-backend-go-clean/internal/testutil/mock"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	humaerr.Setup()
	os.Exit(m.Run())
}

func setupAuthAPI(ctrl *AuthController) (*gin.Engine, huma.API) {
	r := gin.New()
	api := humagin.New(r, huma.DefaultConfig("Test", "1.0.0"))

	huma.Register(api, huma.Operation{
		OperationID: "auth-register",
		Method:      http.MethodPost,
		Path:        "/register",
	}, ctrl.Register)

	huma.Register(api, huma.Operation{
		OperationID: "auth-login",
		Method:      http.MethodPost,
		Path:        "/login",
	}, ctrl.Login)

	huma.Register(api, huma.Operation{
		OperationID: "auth-refresh",
		Method:      http.MethodPost,
		Path:        "/refresh",
	}, ctrl.RefreshToken)

	huma.Register(api, huma.Operation{
		OperationID: "auth-logout",
		Method:      http.MethodPost,
		Path:        "/logout",
	}, ctrl.Logout)

	return r, api
}

func toJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	data, err := json.Marshal(v)
	assert.NoError(t, err)
	return bytes.NewBuffer(data)
}

// ─── Register ─────────────────────────────────────────────────────────────────

func TestAuthController_Register_Success(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	mockUC.On("Register", mock.Anything, mock.AnythingOfType("*entity.RegisterRequest")).Return(
		&entity.RegisterResponse{User: entity.User{ID: 1, Username: "kirk"}}, nil,
	)

	body := toJSON(t, entity.RegisterRequest{Username: "kirk", Email: "kirk@example.com", Password: "securepass"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "kirk")
}

func TestAuthController_Register_Conflict(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	mockUC.On("Register", mock.Anything, mock.AnythingOfType("*entity.RegisterRequest")).Return(
		nil, domainerrors.ErrUsernameExists,
	)

	body := toJSON(t, entity.RegisterRequest{Username: "kirk", Email: "kirk@example.com", Password: "securepass"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// huma.NewError (customized) should extract 409 from ErrUsernameExists
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAuthController_Register_InvalidInput(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── Login ────────────────────────────────────────────────────────────────────

func TestAuthController_Login_Success(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	mockUC.On("Login", mock.Anything, mock.AnythingOfType("*entity.LoginRequest")).Return(
		&entity.LoginResponse{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
			ExpiresAt:    time.Now().Add(time.Hour),
			User:         entity.User{ID: 1, Username: "kirk"},
		}, nil,
	)

	body := toJSON(t, entity.LoginRequest{Username: "kirk", Password: "securepass"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "access-token")
}

func TestAuthController_Login_InvalidCredentials(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	mockUC.On("Login", mock.Anything, mock.AnythingOfType("*entity.LoginRequest")).Return(
		nil, domainerrors.ErrInvalidCredentials,
	)

	body := toJSON(t, entity.LoginRequest{Username: "kirk", Password: "wrong"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── RefreshToken ─────────────────────────────────────────────────────────────

func TestAuthController_RefreshToken_Success(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	mockUC.On("RefreshToken", mock.Anything, mock.AnythingOfType("*entity.RefreshTokenRequest")).Return(
		&entity.RefreshTokenResponse{
			AccessToken:  "new-access",
			RefreshToken: "new-refresh",
			ExpiresAt:    time.Now().Add(time.Hour),
		}, nil,
	)

	body := toJSON(t, entity.RefreshTokenRequest{RefreshToken: "old-refresh"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/refresh", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "new-access")
}

func TestAuthController_RefreshToken_Revoked(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	mockUC.On("RefreshToken", mock.Anything, mock.AnythingOfType("*entity.RefreshTokenRequest")).Return(
		nil, domainerrors.ErrTokenBlacklisted,
	)

	body := toJSON(t, entity.RefreshTokenRequest{RefreshToken: "revoked"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/refresh", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── Logout ───────────────────────────────────────────────────────────────────

func TestAuthController_Logout_Success(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	mockUC.On("Logout", mock.Anything, mock.AnythingOfType("*entity.LogoutRequest")).Return(nil)

	body := toJSON(t, entity.LogoutRequest{RefreshToken: "token"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/logout", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Logged out successfully")
}

// ─── Invalid JSON body tests ──────────────────────────────────────────────────

func TestAuthController_Login_InvalidJSON(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthController_RefreshToken_InvalidJSON(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/refresh", bytes.NewBufferString(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthController_Logout_InvalidJSON(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/logout", bytes.NewBufferString(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthController_Logout_UseCaseError(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	mockUC.On("Logout", mock.Anything, mock.AnythingOfType("*entity.LogoutRequest")).Return(
		domainerrors.ErrInternal,
	)

	body := toJSON(t, entity.LogoutRequest{RefreshToken: "token"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/logout", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
