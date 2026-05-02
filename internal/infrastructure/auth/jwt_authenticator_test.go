package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
)

const (
	testAccessSecret  = "test-access-secret-key-for-testing"
	testRefreshSecret = "test-refresh-secret-key-for-testing"
	testIssuer        = "test-issuer"
)

func newTestAuthenticator() *jwtAuthenticator {
	bl := &TokenBlacklist{
		blacklist: make(map[string]time.Time),
	}
	return NewJWTAuthenticator(
		testAccessSecret, testRefreshSecret, testIssuer,
		15*time.Minute, 24*time.Hour,
		bl,
	).(*jwtAuthenticator)
}

func testUser() *entity.User {
	return &entity.User{
		ID:       42,
		Username: "kirk",
		Email:    "kirk@example.com",
	}
}

// ─── GenerateTokenPair ────────────────────────────────────────────────────────

func TestJWTAuthenticator_GenerateTokenPair_Success(t *testing.T) {
	auth := newTestAuthenticator()

	pair, err := auth.GenerateTokenPair(testUser())

	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.True(t, pair.ExpiresAt.After(time.Now()))
	// Access and refresh tokens should be different
	assert.NotEqual(t, pair.AccessToken, pair.RefreshToken)
}

// ─── ValidateAccessToken ──────────────────────────────────────────────────────

func TestJWTAuthenticator_ValidateAccessToken_Success(t *testing.T) {
	auth := newTestAuthenticator()

	pair, err := auth.GenerateTokenPair(testUser())
	require.NoError(t, err)

	claims, stdClaims, err := auth.ValidateAccessToken(pair.AccessToken)

	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, "kirk", claims.Username)
	assert.Equal(t, testIssuer, stdClaims.Issuer)
	assert.True(t, stdClaims.ExpiresAt > 0)
}

func TestJWTAuthenticator_ValidateAccessToken_InvalidToken(t *testing.T) {
	auth := newTestAuthenticator()

	_, _, err := auth.ValidateAccessToken("invalid.token.string")

	assert.Error(t, err)
}

func TestJWTAuthenticator_ValidateAccessToken_WrongSecret(t *testing.T) {
	// Generate with one authenticator, validate with another using a different secret
	auth1 := newTestAuthenticator()
	bl := &TokenBlacklist{blacklist: make(map[string]time.Time)}
	auth2 := NewJWTAuthenticator(
		"different-access-secret", testRefreshSecret, testIssuer,
		15*time.Minute, 24*time.Hour, bl,
	).(*jwtAuthenticator)

	pair, err := auth1.GenerateTokenPair(testUser())
	require.NoError(t, err)

	_, _, err = auth2.ValidateAccessToken(pair.AccessToken)

	assert.Error(t, err)
}

func TestJWTAuthenticator_ValidateAccessToken_ExpiredToken(t *testing.T) {
	bl := &TokenBlacklist{blacklist: make(map[string]time.Time)}
	auth := NewJWTAuthenticator(
		testAccessSecret, testRefreshSecret, testIssuer,
		1*time.Nanosecond, 24*time.Hour, // Extremely short access expiration
		bl,
	).(*jwtAuthenticator)

	pair, err := auth.GenerateTokenPair(testUser())
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	_, _, err = auth.ValidateAccessToken(pair.AccessToken)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is expired")
}

func TestJWTAuthenticator_ValidateAccessToken_RefreshTokenRejected(t *testing.T) {
	// A refresh token should NOT be valid as an access token (different secrets)
	auth := newTestAuthenticator()

	pair, err := auth.GenerateTokenPair(testUser())
	require.NoError(t, err)

	_, _, err = auth.ValidateAccessToken(pair.RefreshToken)

	assert.Error(t, err, "refresh token should not be accepted as access token")
}

// ─── ValidateRefreshToken ─────────────────────────────────────────────────────

func TestJWTAuthenticator_ValidateRefreshToken_Success(t *testing.T) {
	auth := newTestAuthenticator()

	pair, err := auth.GenerateTokenPair(testUser())
	require.NoError(t, err)

	claims, stdClaims, err := auth.ValidateRefreshToken(pair.RefreshToken)

	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.UserID)
	assert.Equal(t, testIssuer, stdClaims.Issuer)
}

func TestJWTAuthenticator_ValidateRefreshToken_InvalidToken(t *testing.T) {
	auth := newTestAuthenticator()

	_, _, err := auth.ValidateRefreshToken("garbage")

	assert.Error(t, err)
}

func TestJWTAuthenticator_ValidateRefreshToken_AccessTokenRejected(t *testing.T) {
	// An access token should NOT be valid as a refresh token
	auth := newTestAuthenticator()

	pair, err := auth.GenerateTokenPair(testUser())
	require.NoError(t, err)

	_, _, err = auth.ValidateRefreshToken(pair.AccessToken)

	assert.Error(t, err, "access token should not be accepted as refresh token")
}

func TestJWTAuthenticator_ValidateRefreshToken_Expired(t *testing.T) {
	bl := &TokenBlacklist{blacklist: make(map[string]time.Time)}
	auth := NewJWTAuthenticator(
		testAccessSecret, testRefreshSecret, testIssuer,
		15*time.Minute, 1*time.Nanosecond, // Extremely short refresh expiration
		bl,
	).(*jwtAuthenticator)

	pair, err := auth.GenerateTokenPair(testUser())
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	_, _, err = auth.ValidateRefreshToken(pair.RefreshToken)

	assert.Error(t, err)
}

// ─── Blacklist integration ────────────────────────────────────────────────────

func TestJWTAuthenticator_BlacklistToken(t *testing.T) {
	auth := newTestAuthenticator()

	pair, err := auth.GenerateTokenPair(testUser())
	require.NoError(t, err)

	assert.False(t, auth.IsTokenBlacklisted(pair.RefreshToken))

	auth.BlacklistToken(pair.RefreshToken, 1*time.Hour)

	assert.True(t, auth.IsTokenBlacklisted(pair.RefreshToken))
}

func TestJWTAuthenticator_BlacklistToken_NotAffectOtherTokens(t *testing.T) {
	auth := newTestAuthenticator()

	user1 := &entity.User{ID: 1, Username: "alice"}
	user2 := &entity.User{ID: 2, Username: "bob"}

	pair1, _ := auth.GenerateTokenPair(user1)
	pair2, _ := auth.GenerateTokenPair(user2)

	auth.BlacklistToken(pair1.RefreshToken, 1*time.Hour)

	assert.True(t, auth.IsTokenBlacklisted(pair1.RefreshToken))
	assert.False(t, auth.IsTokenBlacklisted(pair2.RefreshToken))
}

// ─── Signing method validation ────────────────────────────────────────────────

func TestJWTAuthenticator_RejectsNoneAlgorithm(t *testing.T) {
	auth := newTestAuthenticator()

	// An unsigned token with "none" algorithm should be rejected
	_, _, err := auth.ValidateAccessToken("eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1c2VyX2lkIjo0MiwidXNlcm5hbWUiOiJoYWNrZXIifQ.")

	assert.Error(t, err)
}
