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
	// 否则可以枚举用户名

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

// ─── 注册时不做邮箱唯一性校验是否是有意的？ ──────────────────────────────────

func TestRegister_DuplicateEmail_IsNotChecked(t *testing.T) {
	// 当前代码只检查 username 唯一性，不检查 email
	// 这意味着不同用户可以用同一个邮箱注册
	// 这可能是 BUG，也可能是设计选择——需要显式记录

	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	repo.On("FindByUsername", mock.Anything, "alice").Return(nil, domainerrors.ErrUserNotFound)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).Return(nil)

	// 即使邮箱 "taken@example.com" 已被其他用户使用，注册仍然成功
	resp, err := uc.Register(context.Background(), &entity.RegisterRequest{
		Username: "alice",
		Email:    "taken@example.com",
		Password: "securepassword",
	})

	// 当前行为：注册成功（因为没有邮箱唯一性检查）
	// 如果未来要加邮箱唯一性检查，这个测试应该改为 assert error
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

// ─── 注册不做输入验证是否是有意的？ ──────────────────────────────────────────

func TestRegister_DoesNotValidateInput(t *testing.T) {
	// Register 没有调用 user.Validate()
	// 这意味着可以注册空用户名、短密码等
	// UpdateUser 调用了 Validate() 但 Register 没有——这是不一致的

	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	repo.On("FindByUsername", mock.Anything, "").Return(nil, domainerrors.ErrUserNotFound)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).Return(nil)

	// 空用户名 + 短密码 → 当前行为是通过（因为没有 Validate）
	resp, err := uc.Register(context.Background(), &entity.RegisterRequest{
		Username: "",
		Email:    "",
		Password: "short",
	})

	// 记录当前行为：Register 不做验证，依赖 Gin binding tags
	// 如果 Register 应该调用 Validate()，这个测试应该改为 assert error
	assert.NoError(t, err, "current behavior: Register does NOT call Validate()")
	assert.NotNil(t, resp)
}

func TestRegister_BcryptRejectsPasswordOver72Bytes(t *testing.T) {
	// Go 1.24+ bcrypt 拒绝超过 72 字节的密码（不再静默截断）
	// 当前代码没有在 Register 层面做密码长度上限校验
	// 导致用户输入超长密码时收到 500 Internal Server Error
	// 这应该在验证层返回 400 Bad Request

	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	longPassword := strings.Repeat("a", 73) // 超过 72 字节

	repo.On("FindByUsername", mock.Anything, "alice").Return(nil, domainerrors.ErrUserNotFound)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).Return(nil).Maybe()

	_, err := uc.Register(context.Background(), &entity.RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: longPassword,
	})

	// 当前行为：返回 ErrInternal（500），因为 bcrypt.GenerateFromPassword 报错
	// 理想行为：应该在输入验证阶段返回 400
	assert.Error(t, err, "passwords > 72 bytes must be rejected")
	assert.Contains(t, err.Error(), "INTERNAL_ERROR",
		"BUG: should be a validation error (400), not an internal error (500)")
}

func TestRegister_PasswordExactly72BytesWorks(t *testing.T) {
	repo := new(testmock.MockUserRepository)
	auth := new(testmock.MockAuthenticator)
	uc := newAuthUseCase(repo, auth)

	password72 := strings.Repeat("a", 72) // 恰好 72 字节

	repo.On("FindByUsername", mock.Anything, "alice").Return(nil, domainerrors.ErrUserNotFound)
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

// ─── Refresh 后旧 token 是否失效？ ───────────────────────────────────────────

func TestRefreshToken_OldTokenNotBlacklisted(t *testing.T) {
	// 当前实现：RefreshToken 成功后没有把旧的 refresh token 加入黑名单
	// 这意味着旧 token 仍然可用，存在 token 重放攻击风险
	// 这是一个需要明确评估的安全决策

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

	_, err := uc.RefreshToken(context.Background(), &entity.RefreshTokenRequest{
		RefreshToken: "old-refresh",
	})
	assert.NoError(t, err)

	// 旧 token 没有被黑名单——这是当前行为
	// 如果要修复 token rotation，应该在 RefreshToken 中 BlacklistToken("old-refresh", ...)
	auth.AssertNotCalled(t, "BlacklistToken",
		"current behavior: old refresh token is NOT blacklisted after rotation — potential replay risk")
}
