package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetTargetUserIDFromParam extracts the target user ID from the "id" URL parameter.
func GetTargetUserIDFromParam(c *gin.Context) (int64, error) {
	idFromParams := c.Param("id")
	targetUserID, err := strconv.ParseInt(idFromParams, 10, 64)
	if err != nil {
		return 0, err // Invalid URL parameter
	}
	return targetUserID, nil
}
