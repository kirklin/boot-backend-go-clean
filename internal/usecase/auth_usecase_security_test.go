package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity"
	domainerrors "github.com/kirklin/boot-backend-go-clean/internal/domain/errors"
	testmock "github.com/kirklin/boot-backend-go-clean/internal/testutil/mock"
)

// ─── 密码绝不能以明文存储 ─────────────────────────────────────────────────────

func TestRegister_PasswordIsHashed_NotPlaintext(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	var capturedUser *entity.User
	repo.On("FindByUsername", mock.Anything, "alice").Return(nil, domainerrors.ErrUserNotFound)
	repo.On("FindByEmail", mock.Anything, "alice@example.com").Return(nil, domainerrors.ErrUserNotFound)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).
		Run(func(args mock.Arguments) {
			capturedUser = args.Get(1).(*entity.User)
		}).Return(nil)

	_, err := uc.Register(context.Background(), &entity.RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "myplainpassword",
	})

	assert.NoError(t, err)
	// 核心断言：存入 DB 的密码绝不能是明文
	assert.NotEqual(t, "myplainpassword", capturedUser.Password,
		"CRITICAL: password stored in plaintext!")
	// 必须是合法的 bcrypt hash
	assert.NoError(t, bcrypt.CompareHashAndPassword(
		[]byte(capturedUser.Password), []byte("myplainpassword")),
		"stored password should be a valid bcrypt hash of the original")
}

// ─── 密码绝不能出现在 API 响应中 ──────────────────────────────────────────────

func TestRegister_ResponseNeverLeaksPassword(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	repo.On("FindByUsername", mock.Anything, "alice").Return(nil, domainerrors.ErrUserNotFound)
	repo.On("FindByEmail", mock.Anything, "alice@example.com").Return(nil, domainerrors.ErrUserNotFound)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).Return(nil)

	resp, err := uc.Register(context.Background(), &entity.RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "supersecret123",
	})

	assert.NoError(t, err)

	// 把响应序列化成 JSON，模拟真实 HTTP 响应
	jsonBytes, _ := json.Marshal(resp)
	jsonStr := string(jsonBytes)

	// json:"-" tag 应该阻止密码出现在 JSON 中
	assert.NotContains(t, jsonStr, "supersecret123",
		"CRITICAL: plaintext password leaked in response!")
	assert.NotContains(t, jsonStr, "$2a$",
		"CRITICAL: bcrypt hash leaked in response!")
}

func TestLogin_ResponseNeverLeaksPassword(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	hashedPw, _ := bcryptHash("correctpassword")
	user := &entity.User{ID: 1, Username: "kirk", Password: hashedPw, Email: "k@example.com"}
	repo.On("FindByUsername", mock.Anything, "kirk").Return(user, nil)
	auth.On("GenerateTokenPair", user).Return(&entity.TokenPair{
		AccessToken: "at", RefreshToken: "rt",
	}, nil)

	resp, err := uc.Login(context.Background(), &entity.LoginRequest{
		Username: "kirk", Password: "correctpassword",
	})

	assert.NoError(t, err)

	jsonBytes, _ := json.Marshal(resp)
	jsonStr := string(jsonBytes)

	assert.NotContains(t, jsonStr, "correctpassword",
		"CRITICAL: plaintext password in login response!")
	assert.NotContains(t, jsonStr, "$2a$",
		"CRITICAL: bcrypt hash leaked in login response!")
}

// ─── 登录失败时不能泄露用户是否存在 ──────────────────────────────────────────

func TestLogin_UserNotFound_And_WrongPassword_ReturnSameError(t *testing.T) {
	// 攻击者不应该能通过错误消息区分 "用户不存在" 和 "密码错误"

	// Case 1: 用户不存在
	repo1 := new(testmock.MockUserRepository)
	auth1 := new(testmock.MockAuthenticator)
	uc1 := newAuthUseCase(repo1, auth1)
	repo1.On("FindByUsername", mock.Anything, "ghost").Return(nil, domainerrors.ErrUserNotFound)

	_, err1 := uc1.Login(context.Background(), &entity.LoginRequest{
		Username: "ghost", Password: "whatever",
	})

	// Case 2: 用户存在但密码错误
	repo2 := new(testmock.MockUserRepository)
	auth2 := new(testmock.MockAuthenticator)
	uc2 := newAuthUseCase(repo2, auth2)
	hashedPw, _ := bcryptHash("correct")
	repo2.On("FindByUsername", mock.Anything, "kirk").Return(
		&entity.User{ID: 1, Username: "kirk", Password: hashedPw}, nil)

	_, err2 := uc2.Login(context.Background(), &entity.LoginRequest{
		Username: "kirk", Password: "wrong",
	})

	// 两种情况返回的错误必须完全一致，攻击者无法区分
	assert.Equal(t, err1.Error(), err2.Error(),
		"error messages must be identical to prevent username enumeration")
}

// ─── 邮箱唯一性校验 ──────────────────────────────────────────────────────────

func TestRegister_DuplicateEmail_IsRejected(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	repo.On("FindByUsername", mock.Anything, "alice").Return(nil, domainerrors.ErrUserNotFound)
	// 邮箱已被其他用户注册
	existingUser := &entity.User{ID: 99, Username: "bob", Email: "taken@example.com"}
	repo.On("FindByEmail", mock.Anything, "taken@example.com").Return(existingUser, nil)

	resp, err := uc.Register(context.Background(), &entity.RegisterRequest{
		Username: "alice",
		Email:    "taken@example.com",
		Password: "securepassword1",
	})

	assert.ErrorIs(t, err, domainerrors.ErrEmailExists)
	assert.Nil(t, resp)
}

// ─── 注册输入验证 ────────────────────────────────────────────────────────────

func TestRegister_ValidatesInput(t *testing.T) {
	// Register 现在调用 Validate()，空用户名应被拒绝

	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	resp, err := uc.Register(context.Background(), &entity.RegisterRequest{
		Username: "",
		Email:    "",
		Password: "short",
	})

	assert.Error(t, err, "empty username should be rejected by Validate()")
	assert.Nil(t, resp)
}

// ─── 密码长度上限 ────────────────────────────────────────────────────────────

func TestRegister_PasswordOver72Bytes_ReturnsBadRequest(t *testing.T) {
	// Domain 层 Validate() 现在检查密码长度上限
	// 确保返回 400 而不是 500

	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	longPassword := strings.Repeat("a", 73)

	_, err := uc.Register(context.Background(), &entity.RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: longPassword,
	})

	assert.Error(t, err, "passwords > 72 bytes must be rejected")
	assert.Contains(t, err.Error(), "VALIDATION_FAILED",
		"should be a validation error (400), not internal (500)")
}

func TestRegister_PasswordExactly72BytesWorks(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	password72 := strings.Repeat("a", 72)

	repo.On("FindByUsername", mock.Anything, "alice").Return(nil, domainerrors.ErrUserNotFound)
	repo.On("FindByEmail", mock.Anything, "alice@example.com").Return(nil, domainerrors.ErrUserNotFound)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).Return(nil)

	resp, err := uc.Register(context.Background(), &entity.RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: password72,
	})

	assert.NoError(t, err, "72-byte password should be accepted (bcrypt max)")
	assert.NotNil(t, resp)
}

// ─── Logout 后 token 应该立即失效 ────────────────────────────────────────────

func TestLogout_TokenIsBlacklistedImmediately(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	var blacklistedToken string
	auth.On("BlacklistToken", mock.AnythingOfType("string"), mock.Anything).
		Run(func(args mock.Arguments) {
			blacklistedToken = args.Get(0).(string)
		}).Return()

	err := uc.Logout(context.Background(), &entity.LogoutRequest{
		RefreshToken: "token-to-revoke",
	})

	assert.NoError(t, err)
	assert.Equal(t, "token-to-revoke", blacklistedToken,
		"the exact token from the request must be blacklisted")
}

// ─── Refresh 后旧 token 必须失效 ─────────────────────────────────────────────

func TestRefreshToken_OldTokenIsBlacklisted(t *testing.T) {
	// 刷新成功后旧 refresh token 应被加入黑名单，防止重放攻击

	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	auth.On("IsTokenBlacklisted", "old-refresh").Return(false)
	auth.On("ValidateRefreshToken", "old-refresh").Return(
		&entity.RefreshTokenClaims{UserID: 1}, &entity.StandardClaims{}, nil,
	)
	user := &entity.User{ID: 1, Username: "kirk"}
	repo.On("FindByID", mock.Anything, int64(1)).Return(user, nil)
	auth.On("GenerateTokenPair", user).Return(&entity.TokenPair{
		AccessToken: "new-at", RefreshToken: "new-rt",
	}, nil)
	auth.On("BlacklistToken", "old-refresh", mock.Anything).Return()

	_, err := uc.RefreshToken(context.Background(), &entity.RefreshTokenRequest{
		RefreshToken: "old-refresh",
	})
	assert.NoError(t, err)

	// 验证旧 token 确实被加入了黑名单
	auth.AssertCalled(t, "BlacklistToken", "old-refresh", mock.Anything)
}
