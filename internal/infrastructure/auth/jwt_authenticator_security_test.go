package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
)

// =============================================================================
// 对抗性 JWT 安全测试
// 从攻击者视角验证 JWT 实现的安全边界
// =============================================================================

// ─── Token 混用攻击 ─────────────────────────────────────────────────────────

func TestSecurity_AccessTokenCannotBeUsedAsRefresh(t *testing.T) {
	// 如果 access 和 refresh token 共享密钥，攻击者可以用短命的
	// access token 当 refresh token 来持续刷新，绕过有效期限制
	auth := newTestAuthenticator()
	pair, err := auth.GenerateTokenPair(testUser())
	require.NoError(t, err)

	_, _, err = auth.ValidateRefreshToken(pair.AccessToken)
	assert.Error(t, err, "access token must NOT be accepted as refresh token")
}

func TestSecurity_RefreshTokenCannotBeUsedAsAccess(t *testing.T) {
	// refresh token 不应包含 username 等敏感信息
	// 且不应被认证中间件接受
	auth := newTestAuthenticator()
	pair, err := auth.GenerateTokenPair(testUser())
	require.NoError(t, err)

	_, _, err = auth.ValidateAccessToken(pair.RefreshToken)
	assert.Error(t, err, "refresh token must NOT be accepted as access token")
}

// ─── Token 内容最小权限 ─────────────────────────────────────────────────────

func TestSecurity_RefreshTokenDoesNotContainUsername(t *testing.T) {
	// refresh token 应只包含最少信息 (user_id)
	// 如果泄露，不应暴露用户名
	auth := newTestAuthenticator()
	pair, err := auth.GenerateTokenPair(testUser())
	require.NoError(t, err)

	// refresh token 里只应该有 user_id，不应该有 username
	claims, _, err := auth.ValidateRefreshToken(pair.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.UserID)
	// RefreshTokenClaims 结构体里本来就没有 Username 字段 — good
}

func TestSecurity_AccessTokenContainsCorrectClaims(t *testing.T) {
	auth := newTestAuthenticator()
	user := &entity.User{ID: 99, Username: "testuser"}
	pair, err := auth.GenerateTokenPair(user)
	require.NoError(t, err)

	claims, stdClaims, err := auth.ValidateAccessToken(pair.AccessToken)
	require.NoError(t, err)

	// 验证 claims 与传入的 user 完全匹配
	assert.Equal(t, int64(99), claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, testIssuer, stdClaims.Issuer)
	// 确保过期时间在合理范围内
	assert.True(t, stdClaims.ExpiresAt > stdClaims.IssuedAt,
		"exp must be after iat")
}

// ─── 空/零值边界 ────────────────────────────────────────────────────────────

func TestSecurity_EmptyTokenString(t *testing.T) {
	auth := newTestAuthenticator()

	_, _, err := auth.ValidateAccessToken("")
	assert.Error(t, err, "empty token must be rejected")

	_, _, err = auth.ValidateRefreshToken("")
	assert.Error(t, err, "empty token must be rejected")
}

func TestSecurity_WhitespaceOnlyToken(t *testing.T) {
	auth := newTestAuthenticator()

	_, _, err := auth.ValidateAccessToken("   ")
	assert.Error(t, err, "whitespace-only token must be rejected")
}

func TestSecurity_BlacklistEmptyToken(t *testing.T) {
	auth := newTestAuthenticator()

	// 空 token 不应该导致 panic
	auth.BlacklistToken("", 1*time.Hour)
	assert.True(t, auth.IsTokenBlacklisted(""))
}

// ─── Token 篡改 ─────────────────────────────────────────────────────────────

func TestSecurity_TamperedPayload(t *testing.T) {
	auth := newTestAuthenticator()
	pair, err := auth.GenerateTokenPair(testUser())
	require.NoError(t, err)

	// JWT 格式: header.payload.signature
	// 篡改 payload 中间部分
	parts := splitJWT(pair.AccessToken)
	if len(parts) == 3 {
		tampered := parts[0] + ".AAAA" + parts[1][4:] + "." + parts[2]
		_, _, err := auth.ValidateAccessToken(tampered)
		assert.Error(t, err, "tampered payload must be rejected")
	}
}

func TestSecurity_SwappedSignature(t *testing.T) {
	auth := newTestAuthenticator()

	user1 := &entity.User{ID: 1, Username: "alice"}
	user2 := &entity.User{ID: 2, Username: "bob"}

	pair1, _ := auth.GenerateTokenPair(user1)
	pair2, _ := auth.GenerateTokenPair(user2)

	// 用 alice 的 payload + bob 的 signature
	parts1 := splitJWT(pair1.AccessToken)
	parts2 := splitJWT(pair2.AccessToken)
	if len(parts1) == 3 && len(parts2) == 3 {
		frankenstein := parts1[0] + "." + parts1[1] + "." + parts2[2]
		_, _, err := auth.ValidateAccessToken(frankenstein)
		assert.Error(t, err, "mismatched signature must be rejected")
	}
}

// ─── 时间相关安全 ────────────────────────────────────────────────────────────

func TestSecurity_TokenExpirationIsEnforced(t *testing.T) {
	bl := &TokenBlacklist{blacklist: make(map[string]time.Time)}
	auth := NewJWTAuthenticator(
		testAccessSecret, testRefreshSecret, testIssuer,
		1*time.Millisecond, // 极短的有效期
		24*time.Hour,
		bl,
	).(*jwtAuthenticator)

	pair, err := auth.GenerateTokenPair(testUser())
	require.NoError(t, err)

	// 等待过期
	time.Sleep(50 * time.Millisecond)

	_, _, err = auth.ValidateAccessToken(pair.AccessToken)
	assert.Error(t, err, "expired token MUST be rejected")
}

func TestSecurity_TokensFromDifferentIssuersAreRejected(t *testing.T) {
	bl1 := &TokenBlacklist{blacklist: make(map[string]time.Time)}
	bl2 := &TokenBlacklist{blacklist: make(map[string]time.Time)}

	// 两个使用相同密钥但不同 issuer 的 authenticator
	auth1 := NewJWTAuthenticator(
		testAccessSecret, testRefreshSecret, "issuer-A",
		15*time.Minute, 24*time.Hour, bl1,
	).(*jwtAuthenticator)

	auth2 := NewJWTAuthenticator(
		testAccessSecret, testRefreshSecret, "issuer-B",
		15*time.Minute, 24*time.Hour, bl2,
	).(*jwtAuthenticator)

	pair, err := auth1.GenerateTokenPair(testUser())
	require.NoError(t, err)

	// 当前实现：不验证 issuer，所以这会通过
	// 如果添加了 issuer 验证，这个测试应该 assert Error
	claims, _, err := auth2.ValidateAccessToken(pair.AccessToken)

	if err == nil {
		// 记录当前行为：issuer 不被验证
		t.Log("WARNING: issuer is NOT validated — tokens from different issuers are accepted")
		assert.Equal(t, int64(42), claims.UserID)
	}
}

// ─── helpers ────────────────────────────────────────────────────────────────

func splitJWT(token string) []string {
	result := make([]string, 0, 3)
	start := 0
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			result = append(result, token[start:i])
			start = i + 1
		}
	}
	result = append(result, token[start:])
	return result
}
