package errors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppError_Error(t *testing.T) {
	t.Run("without underlying error", func(t *testing.T) {
		err := &AppError{Code: "TEST_CODE", Message: "something broke", HTTPCode: 500}
		assert.Equal(t, "[TEST_CODE] something broke", err.Error())
	})

	t.Run("with underlying error", func(t *testing.T) {
		cause := fmt.Errorf("db connection refused")
		err := &AppError{Code: "DB_ERROR", Message: "database failure", HTTPCode: 500, Err: cause}
		assert.Equal(t, "[DB_ERROR] database failure: db connection refused", err.Error())
	})
}

func TestAppError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := &AppError{Code: "TEST", Message: "wrapper", HTTPCode: 500, Err: cause}

	assert.Equal(t, cause, errors.Unwrap(err))
}

func TestAppError_Wrap(t *testing.T) {
	original := ErrUserNotFound
	cause := fmt.Errorf("sql: no rows in result set")

	wrapped := original.Wrap(cause)

	// The wrapped error should carry the underlying error
	assert.ErrorIs(t, wrapped, cause)
	assert.Equal(t, original.Code, wrapped.Code)
	assert.Equal(t, original.Message, wrapped.Message)
	assert.Equal(t, original.HTTPCode, wrapped.HTTPCode)

	// Original should NOT be mutated
	assert.Nil(t, original.Err)
}

func TestAppError_WithMessage(t *testing.T) {
	original := ErrBadRequest
	customized := original.WithMessage("username must not contain spaces")

	assert.Equal(t, "username must not contain spaces", customized.Message)
	assert.Equal(t, original.Code, customized.Code)
	assert.Equal(t, original.HTTPCode, customized.HTTPCode)

	// Original should NOT be mutated
	assert.Equal(t, "Invalid request", original.Message)
}

func TestAppError_ErrorsIs(t *testing.T) {
	// Wrap preserves identity through errors.Is
	cause := fmt.Errorf("some cause")
	wrapped := ErrInternal.Wrap(cause)

	assert.True(t, errors.Is(wrapped, cause))
}

func TestAppError_ErrorsAs(t *testing.T) {
	cause := fmt.Errorf("some cause")
	wrapped := ErrUserNotFound.Wrap(cause)

	var appErr *AppError
	require.True(t, errors.As(wrapped, &appErr))
	assert.Equal(t, "USER_NOT_FOUND", appErr.Code)
	assert.Equal(t, http.StatusNotFound, appErr.HTTPCode)
}

func TestSentinelErrors_HTTPCodes(t *testing.T) {
	tests := []struct {
		err      *AppError
		wantHTTP int
		wantCode string
	}{
		{ErrInternal, http.StatusInternalServerError, "INTERNAL_ERROR"},
		{ErrBadRequest, http.StatusBadRequest, "BAD_REQUEST"},
		{ErrNotFound, http.StatusNotFound, "NOT_FOUND"},
		{ErrUnauthorized, http.StatusUnauthorized, "UNAUTHORIZED"},
		{ErrForbidden, http.StatusForbidden, "FORBIDDEN"},
		{ErrConflict, http.StatusConflict, "CONFLICT"},
		{ErrTooManyRequests, http.StatusTooManyRequests, "RATE_LIMITED"},
		{ErrRequestTimeout, http.StatusRequestTimeout, "REQUEST_TIMEOUT"},
		{ErrUsernameExists, http.StatusConflict, "USERNAME_ALREADY_EXISTS"},
		{ErrEmailExists, http.StatusConflict, "EMAIL_ALREADY_EXISTS"},
		{ErrInvalidCredentials, http.StatusUnauthorized, "INVALID_CREDENTIALS"},
		{ErrTokenBlacklisted, http.StatusUnauthorized, "TOKEN_REVOKED"},
		{ErrTokenInvalid, http.StatusUnauthorized, "TOKEN_INVALID"},
		{ErrUserNotFound, http.StatusNotFound, "USER_NOT_FOUND"},
		{ErrValidationFailed, http.StatusBadRequest, "VALIDATION_FAILED"},
		{ErrNoRowsAffected, http.StatusNotFound, "NO_ROWS_AFFECTED"},
		{ErrPermissionDenied, http.StatusForbidden, "PERMISSION_DENIED"},
	}

	for _, tt := range tests {
		t.Run(tt.wantCode, func(t *testing.T) {
			assert.Equal(t, tt.wantHTTP, tt.err.HTTPCode)
			assert.Equal(t, tt.wantCode, tt.err.Code)
			assert.NotEmpty(t, tt.err.Message)
		})
	}
}
