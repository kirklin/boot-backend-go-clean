package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity/response"
	"github.com/kirklin/boot-backend-go-clean/internal/domain/repository"
	"net/http"
)

// EnsureSelfMiddleware ensures that the current user can only operate on their own data
func EnsureSelfMiddleware(getTargetUserId func(c *gin.Context) (int64, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 JWT 中提取已认证用户的 UserID
		authenticatedUserID, exists := GetUserIDFromContext(c)
		if !exists {
			c.JSON(http.StatusForbidden, response.NewErrorResponse("Unable to identify authenticated user", nil))
			c.Abort()
			return
		}

		// 获取当前请求中与目标用户相关的用户 ID
		targetUserID, err := getTargetUserId(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.NewErrorResponse("Invalid target user ID", err))
			c.Abort()
			return
		}

		// 验证是否是本人
		if authenticatedUserID != targetUserID {
			c.JSON(http.StatusForbidden, response.NewErrorResponse(repository.ErrPermissionDenied.Error(), nil))
			c.Abort()
			return
		}

		// 本人操作，允许继续
		c.Next()
	}
}
