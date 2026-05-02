# 单元测试说明文档

本文档描述了项目的单元测试体系，包括测试结构、Mock 机制、用例清单和运行方法。

---

## 目录结构

```
internal/
├── testutil/mock/                          # 集中式 Mock 实现
│   ├── user_repository.go                  # Mock: repository.UserRepository
│   ├── authenticator.go                    # Mock: gateway.Authenticator
│   ├── auth_usecase.go                     # Mock: usecase.AuthUseCase
│   └── user_usecase.go                     # Mock: usecase.UserUseCase
│
├── domain/entity/
│   └── user_test.go                        # User 实体验证测试
├── domain/errors/
│   └── errors_test.go                      # AppError 类型测试
├── domain/entity/response/
│   └── response_test.go                    # API 响应构建测试
│
├── usecase/
│   ├── auth_usecase_test.go                # 认证业务逻辑测试
│   ├── auth_usecase_security_test.go       # 认证安全不变量测试
│   └── user_usecase_test.go                # 用户业务逻辑测试
│
├── interfaces/http/controller/
│   ├── auth_controller_test.go             # 认证 HTTP 端点测试
│   ├── user_controller_test.go             # 用户 HTTP 端点测试
│   └── security_test.go                    # HTTP 层安全对抗性测试
│
├── interfaces/http/middleware/
│   ├── error_handler_test.go               # 错误处理中间件测试
│   ├── jwt_auth_middleware_test.go          # JWT 认证中间件测试
│   ├── ensure_self_middleware_test.go       # 权限校验中间件测试
│   └── limit_middleware_test.go            # 速率限制中间件测试
│
└── infrastructure/auth/
    ├── blacklist_test.go                   # Token 黑名单并发测试
    ├── jwt_authenticator_test.go           # JWT 签发/验证/过期测试
    └── jwt_authenticator_security_test.go  # JWT 安全对抗性测试
```

---

## 测试框架与工具

| 工具 | 作用 |
|------|------|
| Go 标准 `testing` 包 | 测试入口 |
| `github.com/stretchr/testify/assert` | 断言工具 |
| `github.com/stretchr/testify/mock` | 接口 Mock |
| `net/http/httptest` | HTTP 端点集成测试 |
| `go test -race` | 并发安全检测 |

---

## 用例清单

### 1. Domain Layer — `entity/user_test.go`

| 用例 | 说明 | 验证点 |
|------|------|--------|
| `TestUser_Validate/valid_user` | 合法用户通过验证 | 无错误返回 |
| `TestUser_Validate/empty_username` | 用户名为空 | 返回 "username cannot be empty" |
| `TestUser_Validate/empty_email` | 邮箱为空 | 返回 "email cannot be empty" |
| `TestUser_Validate/invalid_email_format` | 邮箱格式非法 | 返回 "invalid email format" |
| `TestUser_Validate/short_password` | 密码不足 8 位 | 返回 "password must be at least 8 characters long" |
| `TestUser_Validate/password_exactly_8_chars` | 密码恰好 8 位（边界） | 无错误返回 |
| `TestUser_Validate/password_exceeds_72_bytes` | 密码超过 72 字节（bcrypt 上限） | 返回 "password must not exceed 72 bytes" |
| `TestUser_Validate/password_exactly_72_bytes` | 密码恰好 72 字节（边界） | 无错误返回 |
| `TestIsValidEmail/*` | 多种邮箱格式验证 | 正确判断合法/非法邮箱 |

### 2. Domain Layer — `errors/errors_test.go`

| 用例 | 说明 | 验证点 |
|------|------|--------|
| `TestAppError_Error/without_underlying` | 无底层错误的 Error() 输出 | `[CODE] message` 格式 |
| `TestAppError_Error/with_underlying` | 有底层错误的 Error() 输出 | `[CODE] message: cause` 格式 |
| `TestAppError_Unwrap` | Unwrap() 返回底层错误 | `errors.Unwrap()` 兼容 |
| `TestAppError_Wrap` | Wrap() 不可变性 | 原始错误不被修改，clone 携带 cause |
| `TestAppError_WithMessage` | WithMessage() 不可变性 | 原始 Message 不被修改 |
| `TestAppError_ErrorsIs` | `errors.Is()` 兼容性 | Wrapped 错误可穿透查找 |
| `TestAppError_ErrorsAs` | `errors.As()` 兼容性 | 可提取 AppError 结构体 |
| `TestSentinelErrors_HTTPCodes` | 所有哨兵错误的 HTTP 状态码 | 17 个错误码正确映射（含 `ErrEmailExists`） |

### 3. Domain Layer — `response/response_test.go`

| 用例 | 说明 | 验证点 |
|------|------|--------|
| `TestNewErrorResponse_WithAppError` | AppError 自动提取 code/message | 不泄露内部错误 |
| `TestNewErrorResponse_WithWrappedAppError` | Wrap 后的 AppError | 底层 DB 错误不泄露 |
| `TestNewErrorResponse_WithGenericError` | 非 AppError 的普通错误 | 使用调用方提供的 message |
| `TestNewErrorResponse_WithNilError` | nil 错误 | 回退到 INTERNAL_ERROR |
| `TestHTTPCodeFromError/*` | HTTP 状态码提取 | AppError → 正确 HTTP 码；非 AppError → fallback |
| `TestNewSuccessResponse` | 成功响应构建 | status=success, data 正确 |
| `TestNewPageResponse` | 分页响应 | pagination 计算正确 |
| `TestNewPageResponse_HasNext` | hasNext 判断 | current*pageSize < total 时为 true |
| `TestResponse_JSON` | JSON 序列化 | 输出包含 status 和 data |

### 4. Usecase Layer — `auth_usecase_test.go`

| 用例 | 说明 | 验证点 |
|------|------|--------|
| `TestAuthUseCase_Register_Success` | 正常注册 | 用户创建成功，密码被 bcrypt 哈希 |
| `TestAuthUseCase_Register_UsernameExists` | 用户名已存在 | 返回 `ErrUsernameExists` (409) |
| `TestAuthUseCase_Register_DBErrorOnFindByUsername` | FindByUsername 返回 DB 错误 | 返回 AppError 包装的内部错误 |
| `TestAuthUseCase_Register_CreateFails` | Create 持久化失败 | 返回内部错误 |
| `TestAuthUseCase_Login_Success` | 正常登录 | 返回 token pair |
| `TestAuthUseCase_Login_UserNotFound` | 用户不存在 | 返回 `ErrInvalidCredentials` (401)，不泄露"用户不存在" |
| `TestAuthUseCase_Login_WrongPassword` | 密码错误 | 返回 `ErrInvalidCredentials` (401) |
| `TestAuthUseCase_Login_DBError` | FindByUsername 返回非用户未找到的 DB 错误 | 返回内部错误，非 ErrInvalidCredentials |
| `TestAuthUseCase_Login_GenerateTokenPairFails` | Token 签发失败 | 返回内部错误 |
| `TestAuthUseCase_RefreshToken_Success` | 正常刷新 | 返回新 token pair + 旧 token 加入黑名单 |
| `TestAuthUseCase_RefreshToken_Blacklisted` | 令牌已吊销 | 返回 `ErrTokenBlacklisted` (401) |
| `TestAuthUseCase_RefreshToken_ValidationFails` | Refresh token 验证失败（篡改） | 返回 TOKEN_INVALID 错误码 |
| `TestAuthUseCase_RefreshToken_UserNotFound` | 用户已被删除 | 返回 `ErrUserNotFound` |
| `TestAuthUseCase_RefreshToken_DBError` | FindByID 返回 DB 错误 | 返回内部错误 |
| `TestAuthUseCase_RefreshToken_GenerateTokenPairFails` | 新 token 签发失败 | 返回内部错误 |
| `TestAuthUseCase_Logout_Success` | 正常登出 | 令牌加入黑名单 |
| `TestAuthUseCase_Logout_CancelledContext` | 上下文已取消 | 返回 `context.Canceled` |

### 4b. Usecase Layer — `auth_usecase_security_test.go`（安全不变量）

| 用例 | 说明 | 验证点 |
|------|------|--------|
| `TestRegister_PasswordIsHashed_NotPlaintext` | 密码绝不以明文存储 | 捕获 Create 参数，断言 bcrypt hash |
| `TestRegister_ResponseNeverLeaksPassword` | 注册响应不泄露密码 | JSON 序列化后不含明文/hash |
| `TestLogin_ResponseNeverLeaksPassword` | 登录响应不泄露密码 | JSON 序列化后不含明文/hash |
| `TestLogin_UserNotFound_And_WrongPassword_ReturnSameError` | 防用户名枚举 | 两种失败返回完全相同的错误信息 |
| `TestRegister_DuplicateEmail_IsRejected` | 邮箱唯一性检查 | 返回 `ErrEmailExists` (409) |
| `TestRegister_ValidatesInput` | Register 调用 Validate() | 空用户名被拒绝 |
| `TestRegister_PasswordOver72Bytes_ReturnsBadRequest` | 密码超 72 字节 | 返回 `VALIDATION_FAILED` (400) 而非 500 |
| `TestRegister_PasswordExactly72BytesWorks` | 密码恰好 72 字节 | 注册成功 |
| `TestLogout_TokenIsBlacklistedImmediately` | 登出后 token 立即失效 | BlacklistToken 被调用且参数正确 |
| `TestRefreshToken_OldTokenIsBlacklisted` | 刷新后旧 token 失效 | 旧 refresh token 加入黑名单，防重放攻击 |

### 5. Usecase Layer — `user_usecase_test.go`

| 用例 | 说明 | 验证点 |
|------|------|--------|
| `TestUserUseCase_GetUserByID_Success` | 按 ID 查询用户 | 返回正确用户 |
| `TestUserUseCase_GetUserByID_NotFound` | 用户不存在 | 返回 `ErrUserNotFound` (404) |
| `TestUserUseCase_UpdateUser_Success` | 更新用户信息 | 验证通过 → FindByID → Update |
| `TestUserUseCase_UpdateUser_ValidationFails` | 验证失败短路 | 不调用 FindByID 和 Update |
| `TestUserUseCase_UpdateUser_NotFound` | 更新不存在的用户 | FindByID 失败后不调用 Update |
| `TestUserUseCase_UpdateUser_UpdateFails` | Update 持久化失败 | 返回 DB 错误 |
| `TestUserUseCase_SoftDeleteUser_Success` | 软删除用户 | FindByID → SoftDelete |
| `TestUserUseCase_SoftDeleteUser_NotFound` | 删除不存在的用户 | 不调用 SoftDelete |
| `TestUserUseCase_SoftDeleteUser_DeleteFails` | SoftDelete 持久化失败 | 返回 DB 错误 |

### 6. Controller Layer — `auth_controller_test.go`

| 用例 | 说明 | 验证点 |
|------|------|--------|
| `TestAuthController_Register_Success` | POST /register 成功 | HTTP 201 + 用户信息 |
| `TestAuthController_Register_Conflict` | 用户名重复 | HTTP 409 + `USERNAME_ALREADY_EXISTS` |
| `TestAuthController_Register_InvalidInput` | 请求体非法 JSON | HTTP 400 |
| `TestAuthController_Login_Success` | POST /login 成功 | HTTP 200 + access_token |
| `TestAuthController_Login_InvalidCredentials` | 密码错误 | HTTP 401 + `INVALID_CREDENTIALS` |
| `TestAuthController_Login_InvalidJSON` | 请求体非法 JSON | HTTP 400 |
| `TestAuthController_RefreshToken_Success` | POST /refresh 成功 | HTTP 200 + 新 token |
| `TestAuthController_RefreshToken_Revoked` | 令牌已吊销 | HTTP 401 + `TOKEN_REVOKED` |
| `TestAuthController_RefreshToken_InvalidJSON` | 请求体非法 JSON | HTTP 400 |
| `TestAuthController_Logout_Success` | POST /logout 成功 | HTTP 200 |
| `TestAuthController_Logout_InvalidJSON` | 请求体非法 JSON | HTTP 400 |
| `TestAuthController_Logout_UseCaseError` | Usecase 返回内部错误 | HTTP 500 |

### 7. Controller Layer — `user_controller_test.go`

| 用例 | 说明 | 验证点 |
|------|------|--------|
| `TestUserController_GetUser_Success` | GET /users/:id 成功 | HTTP 200 + 用户数据 |
| `TestUserController_GetUser_NotFound` | 用户不存在 | HTTP 404 + `USER_NOT_FOUND` |
| `TestUserController_GetUser_InvalidID` | ID 格式非法 | HTTP 400 |
| `TestUserController_UpdateUser_Success` | PUT /users/:id 成功 | HTTP 200 |
| `TestUserController_UpdateUser_ValidationFails` | 验证失败 | HTTP 400 |
| `TestUserController_UpdateUser_InvalidJSON` | 请求体非法 JSON | HTTP 400 |
| `TestUserController_DeleteUser_Success` | DELETE /users/:id 成功 | HTTP 200 |
| `TestUserController_DeleteUser_NotFound` | 用户不存在 | HTTP 404 |
| `TestUserController_DeleteUser_InvalidID` | ID 格式非法 | HTTP 400 |
| `TestUserController_GetCurrentUser_Success` | GET /me 成功 | HTTP 200 + 用户数据 |
| `TestUserController_GetCurrentUser_NoAuth` | 未认证 | HTTP 401 |

### 8. Middleware Layer — `error_handler_test.go`

| 用例 | 说明 | 验证点 |
|------|------|--------|
| `TestErrorHandler_WithAppError` | 处理 AppError | 自动提取 HTTP 状态码和错误码 |
| `TestErrorHandler_WithGenericError` | 处理普通 error | 回退到 500 |
| `TestErrorHandler_NoErrors` | 无错误时透传 | 不干预正常响应 |
| `TestErrorHandler_ResponseAlreadyWritten` | 响应已写入 | 不覆盖已有响应 |

### 9. Middleware Layer — `jwt_auth_middleware_test.go`

| 用例 | 说明 | 验证点 |
|------|------|--------|
| `TestJWTAuth_MissingAuthorizationHeader` | 无 Authorization 头 | HTTP 401 |
| `TestJWTAuth_InvalidFormat_NoBearerPrefix` | 非 Bearer 格式 | HTTP 401 |
| `TestJWTAuth_InvalidFormat_TooManyParts` | 头部包含多余部分 | HTTP 401 |
| `TestJWTAuth_InvalidFormat_OnlyBearer` | 只有 "Bearer" 无 token | HTTP 401 |
| `TestJWTAuth_InvalidToken` | Token 验证失败 | HTTP 401 + "Invalid or expired" |
| `TestJWTAuth_ValidToken` | Token 验证成功 | HTTP 200 + context 注入 user_id/username |
| `TestJWTAuth_BearerCaseInsensitive` | Bearer 大小写不敏感 | HTTP 200 |
| `TestGetUserIDFromContext_Missing` | Context 中无 user_id | 返回 0, false |
| `TestGetUsernameFromContext_Missing` | Context 中无 username | 返回 "", false |

### 10. Middleware Layer — `ensure_self_middleware_test.go`

| 用例 | 说明 | 验证点 |
|------|------|--------|
| `TestEnsureSelf_MatchingUser` | 操作自己的资源 | HTTP 200 放行 |
| `TestEnsureSelf_DifferentUser` | 操作他人的资源 | HTTP 403 + `PERMISSION_DENIED` |
| `TestEnsureSelf_NoAuthenticatedUser` | Context 中无认证用户 | HTTP 403 |
| `TestEnsureSelf_InvalidTargetUserID` | 目标用户 ID 无效 | HTTP 400 |

### 11. Middleware Layer — `limit_middleware_test.go`

| 用例 | 说明 | 验证点 |
|------|------|--------|
| `TestRateLimiter_AllowsUnderLimit` | 限制内请求全部通过 | 全部 HTTP 200 |
| `TestRateLimiter_BlocksOverLimit` | 超出限制后拦截 | HTTP 429 |
| `TestRateLimiter_SetsHeaders` | 响应头包含限流信息 | X-RateLimit-* 头正确 |
| `TestRateLimiter_ResetsAfterWindow` | 窗口过期后重置计数 | 重新允许请求 |
| `TestRateLimiter_Cleanup` | 后台清理过期条目 | 过期 IP 被移除 |

### 12. Infrastructure Layer — `blacklist_test.go`

| 用例 | 说明 | 验证点 |
|------|------|--------|
| `TestTokenBlacklist_AddAndCheck` | 添加并查询令牌 | 已添加的返回 true，未添加的返回 false |
| `TestTokenBlacklist_ExpiredToken` | 过期令牌查询 | 返回 false（逻辑上不在黑名单） |
| `TestTokenBlacklist_ConcurrentAccess` | 并发读写 | 无 data race（`go test -race` 通过） |

### 13. Infrastructure Layer — `jwt_authenticator_test.go`

| 用例 | 说明 | 验证点 |
|------|------|--------|
| `TestJWTAuthenticator_GenerateTokenPair_Success` | 生成 Token Pair | Access/Refresh token 非空且不同 |
| `TestJWTAuthenticator_ValidateAccessToken_Success` | 验证有效 access token | 正确提取 UserID、Username、Issuer |
| `TestJWTAuthenticator_ValidateAccessToken_InvalidToken` | 无效 token 字符串 | 返回错误 |
| `TestJWTAuthenticator_ValidateAccessToken_WrongSecret` | 错误密钥验证 | 返回错误 |
| `TestJWTAuthenticator_ValidateAccessToken_ExpiredToken` | 过期 access token | 返回 "token is expired" |
| `TestJWTAuthenticator_ValidateAccessToken_RefreshTokenRejected` | 用 refresh token 冒充 access | 返回错误 |
| `TestJWTAuthenticator_ValidateRefreshToken_Success` | 验证有效 refresh token | 正确提取 UserID |
| `TestJWTAuthenticator_ValidateRefreshToken_InvalidToken` | 无效 token 字符串 | 返回错误 |
| `TestJWTAuthenticator_ValidateRefreshToken_AccessTokenRejected` | 用 access token 冒充 refresh | 返回错误 |
| `TestJWTAuthenticator_ValidateRefreshToken_Expired` | 过期 refresh token | 返回错误 |
| `TestJWTAuthenticator_BlacklistToken` | 黑名单集成 | 加入后查询返回 true |
| `TestJWTAuthenticator_BlacklistToken_NotAffectOtherTokens` | 黑名单隔离性 | 不影响其他用户的 token |
| `TestJWTAuthenticator_RejectsNoneAlgorithm` | 拒绝 "none" 签名算法 | 返回错误 |

### 14. Infrastructure Layer — `jwt_authenticator_security_test.go`（安全对抗性）

| 用例 | 说明 | 验证点 |
|------|------|--------|
| `TestSecurity_AccessTokenCannotBeUsedAsRefresh` | Access token 不可冒充 refresh | ValidateRefreshToken 返回错误 |
| `TestSecurity_RefreshTokenCannotBeUsedAsAccess` | Refresh token 不可冒充 access | ValidateAccessToken 返回错误 |
| `TestSecurity_RefreshTokenDoesNotContainUsername` | Refresh token 最小权限 | 仅含 UserID，不含 Username |
| `TestSecurity_AccessTokenContainsCorrectClaims` | Access token claims 准确性 | UserID/Username/Issuer/过期时间验证 |
| `TestSecurity_EmptyTokenString` | 空 token 字符串 | 两种验证均返回错误 |
| `TestSecurity_WhitespaceOnlyToken` | 空白 token | 返回错误 |
| `TestSecurity_BlacklistEmptyToken` | 黑名单空 token 不 panic | 不崩溃且查询返回 true |
| `TestSecurity_TamperedPayload` | 篡改 payload 字节 | 签名验证失败 |
| `TestSecurity_SwappedSignature` | 交换不同用户的签名 | 签名验证失败 |
| `TestSecurity_TokenExpirationIsEnforced` | 过期 token 拒绝 | 1ms 有效期的 token 50ms 后被拒 |
| `TestSecurity_TokensFromDifferentIssuersAreRejected` | 跨服务 token 拒绝 | issuer-A 签发的 token 被 issuer-B 拒绝 |

### 15. Controller Layer — `security_test.go`（HTTP 安全对抗性）

| 用例 | 说明 | 验证点 |
|------|------|--------|
| `TestHTTP_Register_PasswordNeverInResponseBody` | 注册响应不含密码 | JSON 中无明文/hash |
| `TestHTTP_Login_PasswordNeverInResponseBody` | 登录响应不含密码 | JSON 中无明文/hash |
| `TestHTTP_Register_InternalErrorNeverLeaksDBDetails` | 错误不泄露 DB 信息 | 响应中无 sql/pq/constraint |
| `TestHTTP_Register_OversizedUsernameHandledGracefully` | 10KB 用户名 | 不 panic，不返回 5xx |
| `TestHTTP_ErrorResponse_AlwaysHasStructuredFormat` | 错误响应结构一致 | 含 status/message/error.code |
| `TestHTTP_SuccessResponse_AlwaysHasStructuredFormat` | 成功响应结构一致 | 含 status/data，无 error |
| `TestHTTP_Login_WithoutContentType_StillReturnsJSON` | 无 Content-Type | 400 且响应仍为 JSON |
| `TestHTTP_Login_EmptyBody_Returns400` | 空请求体 | 400 |
| `TestHTTP_GetUser_ZeroID` | ID 为 0 | 400 或 404 |
| `TestHTTP_GetUser_NegativeID` | 负数 ID | 400 或 404 |
| `TestHTTP_GetUser_MaxInt64ID` | int64 最大值 | 不 panic |
| `TestHTTP_GetUser_OverflowID` | 超出 int64 范围 | 400 |

---

## 运行命令

```bash
# 运行全部测试
go test ./...

# 运行全部测试（详细输出）
go test -v ./...

# 运行特定包的测试
go test ./internal/usecase/...
go test ./internal/interfaces/http/controller/...

# 运行特定测试函数
go test ./internal/usecase/... -run TestAuthUseCase_Login

# 查看覆盖率
go test -cover ./internal/...

# 生成覆盖率 HTML 报告
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out -o coverage.html

# 并发安全检测（Race Detector）
go test -race ./...
```

---

## Mock 使用指南

所有 Mock 位于 `internal/testutil/mock/`，按接口拆分为独立文件。
每个 Mock 都包含编译时接口一致性断言，确保接口变更时编译即失败：

| 文件 | Mock 类型 | 目标接口 | 编译时断言 |
|------|-----------|----------|-----------|
| `user_repository.go` | `MockUserRepository` | `repository.UserRepository` | ✅ |
| `authenticator.go` | `MockAuthenticator` | `gateway.Authenticator` | ✅ |
| `auth_usecase.go` | `MockAuthUseCase` | `usecase.AuthUseCase` | ✅ |
| `user_usecase.go` | `MockUserUseCase` | `usecase.UserUseCase` | ✅ |

### 使用示例

```go
import testmock "github.com/kirklin/boot-backend-go-clean/internal/testutil/mock"

func TestSomething(t *testing.T) {
    repo := new(testmock.MockUserRepository)

    // 设置预期行为
    repo.On("FindByID", mock.Anything, int64(1)).Return(&entity.User{ID: 1}, nil)

    // ... 执行被测逻辑 ...

    // 验证 Mock 被正确调用
    repo.AssertExpectations(t)
}
```

### 添加新 Mock

1. 在 `internal/testutil/mock/` 下创建新文件，如 `order_repository.go`
2. 定义 `MockOrderRepository struct { mock.Mock }`
3. 实现目标接口的所有方法
4. 添加编译时断言：`var _ repository.OrderRepository = (*MockOrderRepository)(nil)`

---

## 覆盖率报告

> 最后更新：2026-05-02

| 层 | 包 | 覆盖率 |
|---|---|--------|
| Domain | `entity` | **100%** |
| Domain | `errors` | **100%** |
| Domain | `response` | **88.9%** |
| Usecase | `usecase` | **96.4%** |
| Controller | `controller` | **97.2%** |
| Middleware | `middleware` | **78.6%** |
| Infrastructure | `auth` | **79.7%** |
