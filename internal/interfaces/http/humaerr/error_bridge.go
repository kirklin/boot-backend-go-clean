// Package humaerr bridges domain AppError into huma's StatusError interface,
// so that returning an AppError from a huma handler automatically sets the
// correct HTTP status code and renders a consistent JSON response.
package humaerr

import (
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	domainerrors "github.com/kirklin/boot-backend-go-clean/internal/domain/errors"
)

// AppStatusError wraps a domain AppError to satisfy huma.StatusError,
// giving huma the ability to derive the correct HTTP status code.
type AppStatusError struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AppStatusError) Error() string {
	return e.Message
}

// GetStatus satisfies huma.StatusError.
func (e *AppStatusError) GetStatus() int {
	return e.Status
}

// FromAppError converts a domain AppError into huma.StatusError.
// If the error is not an AppError, a generic 500 is returned.
func FromAppError(err error) huma.StatusError {
	var appErr *domainerrors.AppError
	if errors.As(err, &appErr) {
		return &AppStatusError{
			Status:  appErr.HTTPCode,
			Code:    appErr.Code,
			Message: appErr.Message,
		}
	}
	return huma.Error500InternalServerError("Internal server error", err)
}

// NewHumaError converts any error to a huma.StatusError.
// For AppError it maps to the correct HTTP status; otherwise 500.
func NewHumaError(fallbackStatus int, msg string, err error) huma.StatusError {
	var appErr *domainerrors.AppError
	if errors.As(err, &appErr) {
		return huma.NewError(appErr.HTTPCode, appErr.Message, err)
	}
	if err != nil {
		return huma.NewError(fallbackStatus, msg, err)
	}
	return huma.NewError(fallbackStatus, msg)
}

// Setup installs a custom huma.NewError that maps domain AppError
// to the correct HTTP status code automatically.
func Setup() {
	origNewError := huma.NewError
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		// Check if any of the provided errors is an AppError. If so,
		// override the status to use the domain-defined HTTP code.
		for _, err := range errs {
			var appErr *domainerrors.AppError
			if errors.As(err, &appErr) {
				status = appErr.HTTPCode
				if msg == "" || msg == http.StatusText(status) {
					msg = appErr.Message
				}
				break
			}
		}
		return origNewError(status, msg, errs...)
	}
}
