// Package errors defines the domain error types and error codes for the application.
// All business errors should be defined here so that the interfaces layer can
// automatically map them to appropriate HTTP status codes.
//
// This package belongs to the domain layer and has ZERO external dependencies.
package errors

import (
	"fmt"
	"net/http"
)

// AppError represents a structured business error with an error code,
// a user-safe message, and a corresponding HTTP status code.
//
// Controllers should use errors.As to extract AppError and automatically
// derive the correct HTTP status code instead of hard-coding it.
type AppError struct {
	// Code is a machine-readable, self-describing error code (e.g. "USERNAME_ALREADY_EXISTS").
	// Follows the UPPER_SNAKE_CASE convention used by Stripe, Google Cloud, and gRPC.
	// Clients can use this for i18n or conditional error handling.
	Code string

	// Message is a user-safe, human-readable description of the error.
	Message string

	// HTTPCode is the HTTP status code that should be returned to the client.
	HTTPCode int

	// Err is the underlying error that caused this AppError (if any).
	// It is never exposed to the client.
	Err error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap supports errors.Is and errors.As on the underlying error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// Wrap returns a copy of the AppError with the underlying error set.
// This preserves the original error chain for logging while keeping
// the user-facing message unchanged.
//
// Usage:
//
//	return domainerrors.ErrUserNotFound.Wrap(err)
func (e *AppError) Wrap(err error) *AppError {
	clone := *e
	clone.Err = err
	return &clone
}

// WithMessage returns a copy with an overridden user-facing message.
func (e *AppError) WithMessage(msg string) *AppError {
	clone := *e
	clone.Message = msg
	return &clone
}

// =============================================================================
// Common / Shared Errors
// =============================================================================

var (
	// ErrInternal is a catch-all for unexpected server-side failures.
	ErrInternal = &AppError{Code: "INTERNAL_ERROR", Message: "Internal server error", HTTPCode: http.StatusInternalServerError}

	// ErrBadRequest indicates malformed or invalid input from the client.
	ErrBadRequest = &AppError{Code: "BAD_REQUEST", Message: "Invalid request", HTTPCode: http.StatusBadRequest}

	// ErrNotFound indicates the requested resource does not exist.
	ErrNotFound = &AppError{Code: "NOT_FOUND", Message: "Resource not found", HTTPCode: http.StatusNotFound}

	// ErrUnauthorized indicates missing or invalid authentication credentials.
	ErrUnauthorized = &AppError{Code: "UNAUTHORIZED", Message: "Unauthorized", HTTPCode: http.StatusUnauthorized}

	// ErrForbidden indicates the user does not have permission for the operation.
	ErrForbidden = &AppError{Code: "FORBIDDEN", Message: "Permission denied", HTTPCode: http.StatusForbidden}

	// ErrConflict indicates a conflict with the current state (e.g. duplicate entry).
	ErrConflict = &AppError{Code: "CONFLICT", Message: "Resource conflict", HTTPCode: http.StatusConflict}

	// ErrTooManyRequests indicates the client has exceeded the rate limit.
	ErrTooManyRequests = &AppError{Code: "RATE_LIMITED", Message: "Rate limit exceeded. Please try again later.", HTTPCode: http.StatusTooManyRequests}

	// ErrRequestTimeout indicates the request took too long to process.
	ErrRequestTimeout = &AppError{Code: "REQUEST_TIMEOUT", Message: "Request timeout", HTTPCode: http.StatusRequestTimeout}
)

// =============================================================================
// Auth Errors
// =============================================================================

var (
	ErrUsernameExists     = &AppError{Code: "USERNAME_ALREADY_EXISTS", Message: "Username already exists", HTTPCode: http.StatusConflict}
	ErrEmailExists        = &AppError{Code: "EMAIL_ALREADY_EXISTS", Message: "Email already exists", HTTPCode: http.StatusConflict}
	ErrInvalidCredentials = &AppError{Code: "INVALID_CREDENTIALS", Message: "Invalid username or password", HTTPCode: http.StatusUnauthorized}
	ErrTokenBlacklisted   = &AppError{Code: "TOKEN_REVOKED", Message: "Token has been revoked", HTTPCode: http.StatusUnauthorized}
	ErrTokenInvalid       = &AppError{Code: "TOKEN_INVALID", Message: "Invalid or expired token", HTTPCode: http.StatusUnauthorized}
	ErrTokenSigningMethod = &AppError{Code: "TOKEN_SIGNING_INVALID", Message: "Unexpected token signing method", HTTPCode: http.StatusUnauthorized}
)

// =============================================================================
// User Errors
// =============================================================================

var (
	ErrUserNotFound = &AppError{Code: "USER_NOT_FOUND", Message: "User not found", HTTPCode: http.StatusNotFound}
)

// =============================================================================
// Validation Errors
// =============================================================================

var (
	ErrValidationFailed = &AppError{Code: "VALIDATION_FAILED", Message: "Validation failed", HTTPCode: http.StatusBadRequest}
)

// =============================================================================
// Repository Errors
// =============================================================================

var (
	ErrNoRowsAffected   = &AppError{Code: "NO_ROWS_AFFECTED", Message: "No rows affected", HTTPCode: http.StatusNotFound}
	ErrPermissionDenied = &AppError{Code: "PERMISSION_DENIED", Message: "Permission denied for this operation", HTTPCode: http.StatusForbidden}
)

// =============================================================================
// Moderation Errors
// =============================================================================

var (
	ErrLocalImageModeration = &AppError{Code: "IMAGE_MODERATION_UNSUPPORTED", Message: "Local environment does not support image content moderation", HTTPCode: http.StatusNotImplemented}
)
