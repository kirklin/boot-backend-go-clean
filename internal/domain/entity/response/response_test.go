package response

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	domainerrors "github.com/kirklin/boot-backend-go-clean/internal/domain/errors"
)

func TestNewErrorResponse_WithAppError(t *testing.T) {
	err := domainerrors.ErrUsernameExists

	resp := NewErrorResponse("Registration failed", err)

	assert.Equal(t, StatusError, resp.Status)
	assert.Equal(t, "Registration failed", resp.Message)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "USERNAME_ALREADY_EXISTS", resp.Error.Code)
	assert.Equal(t, "Username already exists", resp.Error.Message)
}

func TestNewErrorResponse_WithWrappedAppError(t *testing.T) {
	cause := fmt.Errorf("db: unique constraint violated")
	err := domainerrors.ErrUsernameExists.Wrap(cause)

	resp := NewErrorResponse("Registration failed", err)

	assert.Equal(t, "USERNAME_ALREADY_EXISTS", resp.Error.Code)
	// The underlying db error should NOT leak to the client
	assert.Equal(t, "Username already exists", resp.Error.Message)
}

func TestNewErrorResponse_WithGenericError(t *testing.T) {
	err := fmt.Errorf("some internal SQL error")

	resp := NewErrorResponse("Something went wrong", err)

	assert.Equal(t, "INTERNAL_ERROR", resp.Error.Code)
	// Should use the caller-provided message, NOT err.Error()
	assert.Equal(t, "Something went wrong", resp.Error.Message)
}

func TestNewErrorResponse_WithNilError(t *testing.T) {
	resp := NewErrorResponse("Generic failure", nil)

	assert.Equal(t, "INTERNAL_ERROR", resp.Error.Code)
	assert.Equal(t, "Generic failure", resp.Error.Message)
}

func TestHTTPCodeFromError_WithAppError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		fallback int
		want     int
	}{
		{"UserNotFound", domainerrors.ErrUserNotFound, http.StatusInternalServerError, http.StatusNotFound},
		{"WrappedError", domainerrors.ErrInvalidCredentials.Wrap(fmt.Errorf("bcrypt mismatch")), http.StatusInternalServerError, http.StatusUnauthorized},
		{"GenericError", fmt.Errorf("random error"), http.StatusInternalServerError, http.StatusInternalServerError},
		{"NilError", nil, http.StatusInternalServerError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HTTPCodeFromError(tt.err, tt.fallback)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewSuccessResponse(t *testing.T) {
	type Data struct {
		Name string `json:"name"`
	}

	resp := NewSuccessResponse("OK", Data{Name: "kirk"})

	assert.Equal(t, StatusSuccess, resp.Status)
	assert.Equal(t, "OK", resp.Message)
	assert.Equal(t, "kirk", resp.Data.Name)
	assert.Nil(t, resp.Error)
}

func TestNewPageResponse(t *testing.T) {
	items := []string{"a", "b", "c"}
	resp := NewPageResponse("OK", items, 1, 10, 3)

	assert.Equal(t, StatusSuccess, resp.Status)
	assert.Equal(t, int64(3), resp.Data.Pagination.Total)
	assert.False(t, resp.Data.Pagination.HasNext)
	assert.Len(t, resp.Data.List, 3)
}

func TestNewPageResponse_HasNext(t *testing.T) {
	items := []string{"a", "b"}
	resp := NewPageResponse("OK", items, 1, 2, 10)

	assert.True(t, resp.Data.Pagination.HasNext)
}

func TestResponse_JSON(t *testing.T) {
	resp := NewSuccessResponse("OK", map[string]string{"key": "value"})
	bytes, err := resp.JSON()

	assert.NoError(t, err)
	assert.Contains(t, string(bytes), `"status":"success"`)
	assert.Contains(t, string(bytes), `"key":"value"`)
}
