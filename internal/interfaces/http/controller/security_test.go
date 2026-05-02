package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	domainerrors "github.com/kirklin/boot-backend-go-clean/internal/domain/errors"
	testmock "github.com/kirklin/boot-backend-go-clean/internal/testutil/mock"
)

// =============================================================================
// 对抗性 HTTP 测试
// 从攻击者/客户端角度出发，验证 HTTP 响应的安全性
// =============================================================================

// ─── 密码绝不能出现在 HTTP 响应体中 ──────────────────────────────────────────

func TestHTTP_Register_PasswordNeverInResponseBody(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	// 模拟 Register 返回包含密码的用户对象
	// 即使 usecase 返回了 password 字段，HTTP 响应也不应泄露
	mockUC.On("Register", mock.Anything, mock.AnythingOfType("*entity.RegisterRequest")).Return(
		&entity.RegisterResponse{
			User: entity.User{
				ID:       1,
				Username: "alice",
				Email:    "alice@example.com",
				Password: "$2a$10$hashhashhash", // usecase 可能不小心返回了 hash
			},
		}, nil,
	)

	body := toJSON(t, entity.RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "mysecretpassword",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	respBody := w.Body.String()
	assert.NotContains(t, respBody, "mysecretpassword",
		"CRITICAL: plaintext password in HTTP response!")
	assert.NotContains(t, respBody, "$2a$10$",
		"CRITICAL: bcrypt hash leaked in HTTP response!")
	assert.NotContains(t, respBody, "hashhashhash",
		"CRITICAL: password hash fragments in HTTP response!")
}

func TestHTTP_Login_PasswordNeverInResponseBody(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	mockUC.On("Login", mock.Anything, mock.AnythingOfType("*entity.LoginRequest")).Return(
		&entity.LoginResponse{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
			ExpiresAt:    time.Now().Add(time.Hour),
			User: entity.User{
				ID:       1,
				Username: "kirk",
				Password: "$2a$10$realhashhere", // 即使 usecase 泄露了
			},
		}, nil,
	)

	body := toJSON(t, entity.LoginRequest{Username: "kirk", Password: "secret123"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	respBody := w.Body.String()
	assert.NotContains(t, respBody, "secret123")
	assert.NotContains(t, respBody, "$2a$")
	assert.NotContains(t, respBody, "realhashhere")
}

// ─── 错误响应绝不能泄露内部实现细节 ──────────────────────────────────────────

func TestHTTP_Register_InternalErrorNeverLeaksDBDetails(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	// 模拟一个包含 SQL 错误信息的内部错误
	dbError := domainerrors.ErrInternal.Wrap(
		assert.AnError, // 底层错误
	)
	mockUC.On("Register", mock.Anything, mock.AnythingOfType("*entity.RegisterRequest")).Return(
		nil, dbError,
	)

	body := toJSON(t, entity.RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "securepass",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	respBody := w.Body.String()
	// 不应泄露任何数据库信息
	assert.NotContains(t, respBody, "pq:")
	assert.NotContains(t, respBody, "constraint")
	assert.NotContains(t, respBody, "connection refused")
}

// ─── 超大请求体处理 ──────────────────────────────────────────────────────────

func TestHTTP_Register_OversizedUsernameHandledGracefully(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	// 超长用户名 — 10KB
	longUsername := strings.Repeat("a", 10000)
	mockUC.On("Register", mock.Anything, mock.AnythingOfType("*entity.RegisterRequest")).Return(
		&entity.RegisterResponse{User: entity.User{Username: longUsername}}, nil,
	).Maybe()

	body := toJSON(t, entity.RegisterRequest{
		Username: longUsername,
		Email:    "test@example.com",
		Password: "securepass",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// 不应 panic 或返回 5xx
	assert.True(t, w.Code < 500, "server should not crash on oversized input")
}

// ─── HTTP 响应结构一致性 ─────────────────────────────────────────────────────

func TestHTTP_ErrorResponse_AlwaysHasStructuredFormat(t *testing.T) {
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

	// 验证响应是合法 JSON
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err, "error response must be valid JSON")

	// Huma uses RFC 9457 format: { status, title, detail, errors }
	// Verify it's a valid structured error
	assert.NotNil(t, resp["status"], "status field must be present")
	assert.NotNil(t, resp["detail"], "detail field must be present in huma error")
}

func TestHTTP_SuccessResponse_AlwaysHasStructuredFormat(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	mockUC.On("Login", mock.Anything, mock.AnythingOfType("*entity.LoginRequest")).Return(
		&entity.LoginResponse{
			AccessToken:  "at",
			RefreshToken: "rt",
			ExpiresAt:    time.Now().Add(time.Hour),
			User:         entity.User{ID: 1, Username: "kirk"},
		}, nil,
	)

	body := toJSON(t, entity.LoginRequest{Username: "kirk", Password: "correct"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "success", resp["status"])
	assert.NotNil(t, resp["data"], "success response must include data")
}

// ─── Content-Type 验证 ──────────────────────────────────────────────────────

func TestHTTP_Login_WithoutContentType_StillReturnsJSON(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	w := httptest.NewRecorder()
	// 不设置 Content-Type，发送纯文本
	req, _ := http.NewRequest(http.MethodPost, "/login", strings.NewReader(`not json`))
	router.ServeHTTP(w, req)

	// Huma may return 400 or 415 for unsupported content type
	assert.True(t, w.Code >= 400 && w.Code < 500, "should return 4xx error")

	// 响应本身必须仍然是 JSON
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err, "error response must always be valid JSON regardless of input")
}

func TestHTTP_Login_EmptyBody_Returns400(t *testing.T) {
	mockUC := new(testmock.MockAuthUseCase)
	ctrl := NewAuthController(mockUC)
	router, _ := setupAuthAPI(ctrl)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", nil)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── 边界 ID 值 ─────────────────────────────────────────────────────────────

func TestHTTP_GetUser_ZeroID(t *testing.T) {
	mockUC := new(testmock.MockUserUseCase)
	ctrl := NewUserController(mockUC)
	router, _ := setupUserAPI(ctrl)

	mockUC.On("GetUserByID", mock.Anything, int64(0)).Return(
		nil, domainerrors.ErrUserNotFound,
	).Maybe()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/0", nil)
	router.ServeHTTP(w, req)

	// ID 0 应该被当作无效或未找到
	assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound,
		"ID 0 should not return a valid user")
}

func TestHTTP_GetUser_NegativeID(t *testing.T) {
	mockUC := new(testmock.MockUserUseCase)
	ctrl := NewUserController(mockUC)
	router, _ := setupUserAPI(ctrl)

	mockUC.On("GetUserByID", mock.Anything, int64(-1)).Return(
		nil, domainerrors.ErrUserNotFound,
	).Maybe()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/-1", nil)
	router.ServeHTTP(w, req)

	assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound)
}

func TestHTTP_GetUser_MaxInt64ID(t *testing.T) {
	mockUC := new(testmock.MockUserUseCase)
	ctrl := NewUserController(mockUC)
	router, _ := setupUserAPI(ctrl)

	mockUC.On("GetUserByID", mock.Anything, int64(9223372036854775807)).Return(
		nil, domainerrors.ErrUserNotFound,
	).Maybe()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/users/9223372036854775807", nil)
	router.ServeHTTP(w, req)

	// 不应 panic
	assert.True(t, w.Code < 500, "max int64 ID should not cause server error")
}

func TestHTTP_GetUser_OverflowID(t *testing.T) {
	mockUC := new(testmock.MockUserUseCase)
	ctrl := NewUserController(mockUC)
	router, _ := setupUserAPI(ctrl)

	w := httptest.NewRecorder()
	// 超过 int64 范围的数字
	req, _ := http.NewRequest(http.MethodGet, "/users/99999999999999999999", nil)
	router.ServeHTTP(w, req)

	// Huma returns 422 for invalid path params
	assert.True(t, w.Code >= 400 && w.Code < 500, "overflow ID should return 4xx error")
}
