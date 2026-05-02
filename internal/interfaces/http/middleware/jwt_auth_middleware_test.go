package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
)

// ─── Mock ─────────────────────────────────────────────────────────────────────

// mockTokenValidator is a local mock that implements TokenValidator.
type mockTokenValidator struct {
	mock.Mock
}

func (m *mockTokenValidator) ValidateAccessToken(tokenString string) (*entity.AccessTokenClaims, *entity.StandardClaims, error) {
	args := m.Called(tokenString)
	var ac *entity.AccessTokenClaims
	var sc *entity.StandardClaims
	if args.Get(0) != nil {
		ac = args.Get(0).(*entity.AccessTokenClaims)
	}
	if args.Get(1) != nil {
		sc = args.Get(1).(*entity.StandardClaims)
	}
	return ac, sc, args.Error(2)
}

// ─── Helper ───────────────────────────────────────────────────────────────────

func setupJWTRouter(validator TokenValidator) *gin.Engine {
	r := gin.New()
	r.Use(JWTAuthMiddleware(validator))
	r.GET("/protected", func(c *gin.Context) {
		userID, _ := GetUserIDFromContext(c)
		username, _ := GetUsernameFromContext(c)
		c.JSON(http.StatusOK, gin.H{"user_id": userID, "username": username})
	})
	return r
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestJWTAuth_MissingAuthorizationHeader(t *testing.T) {
	v := new(mockTokenValidator)
	router := setupJWTRouter(v)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authorization header is required")
}

func TestJWTAuth_InvalidFormat_NoBearerPrefix(t *testing.T) {
	v := new(mockTokenValidator)
	router := setupJWTRouter(v)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Token some-token")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid authorization header format")
}

func TestJWTAuth_InvalidFormat_TooManyParts(t *testing.T) {
	v := new(mockTokenValidator)
	router := setupJWTRouter(v)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer token extra")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid authorization header format")
}

func TestJWTAuth_InvalidFormat_OnlyBearer(t *testing.T) {
	v := new(mockTokenValidator)
	router := setupJWTRouter(v)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	v := new(mockTokenValidator)
	router := setupJWTRouter(v)

	v.On("ValidateAccessToken", "bad-token").Return(nil, nil, errors.New("token expired"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid or expired token")
	v.AssertExpectations(t)
}

func TestJWTAuth_ValidToken(t *testing.T) {
	v := new(mockTokenValidator)
	router := setupJWTRouter(v)

	v.On("ValidateAccessToken", "good-token").Return(
		&entity.AccessTokenClaims{UserID: 42, Username: "kirk"},
		&entity.StandardClaims{IssuedAt: 1000, ExpiresAt: 9999, Issuer: "test"},
		nil,
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"user_id":42`)
	assert.Contains(t, w.Body.String(), `"username":"kirk"`)
	v.AssertExpectations(t)
}

func TestJWTAuth_BearerCaseInsensitive(t *testing.T) {
	v := new(mockTokenValidator)
	router := setupJWTRouter(v)

	v.On("ValidateAccessToken", "good-token").Return(
		&entity.AccessTokenClaims{UserID: 1, Username: "kirk"},
		&entity.StandardClaims{},
		nil,
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "BEARER good-token")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	v.AssertExpectations(t)
}

// ─── Context helpers ──────────────────────────────────────────────────────────

func TestGetUserIDFromContext_Missing(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	id, exists := GetUserIDFromContext(c)

	assert.False(t, exists)
	assert.Equal(t, int64(0), id)
}

func TestGetUsernameFromContext_Missing(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	username, exists := GetUsernameFromContext(c)

	assert.False(t, exists)
	assert.Equal(t, "", username)
}
