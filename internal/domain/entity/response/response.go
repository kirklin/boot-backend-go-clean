package response

import (
	"encoding/json"
	"errors"

	domainerrors "github.com/kirklin/boot-backend-go-clean/internal/domain/errors"
)

// ResStatus represents the status of the response
type ResStatus string

const (
	StatusSuccess ResStatus = "success"
	StatusError   ResStatus = "error"
)

// Response is a generic structure for all API responses
type Response[T any] struct {
	Status  ResStatus     `json:"status"`
	Message string        `json:"message"`
	Data    T             `json:"data,omitempty"`
	Error   *ErrorDetails `json:"error,omitempty"`
	Meta    *MetaInfo     `json:"meta,omitempty"`
}

// Pagination contains pagination metadata
type Pagination struct {
	Current       int64  `json:"current"`
	PageSize      int64  `json:"page_size"`
	Total         int64  `json:"total"`
	HasNext       bool   `json:"has_next"`
	NextPageToken string `json:"next_page_token,omitempty"`
}

// PageData contains the list data and pagination info
type PageData[T any] struct {
	List       []T        `json:"list"`
	Pagination Pagination `json:"pagination"`
	Extra      any        `json:"extra,omitempty"`
}

// ErrorDetails contains detailed error information
type ErrorDetails struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

// MetaInfo contains metadata for the API response
type MetaInfo struct {
	Debug string `json:"debug,omitempty"` // Optional debugging info
}

// NewSuccessResponse creates a new success response
func NewSuccessResponse[T any](message string, data T) Response[T] {
	return Response[T]{
		Status:  StatusSuccess,
		Message: message,
		Data:    data,
	}
}

// NewPageResponse creates a new paginated success response
func NewPageResponse[T any](message string, list []T, current, pageSize, total int64) Response[PageData[T]] {
	hasNext := current*pageSize < total

	return Response[PageData[T]]{
		Status:  StatusSuccess,
		Message: message,
		Data: PageData[T]{
			List: list,
			Pagination: Pagination{
				Current:  current,
				PageSize: pageSize,
				Total:    total,
				HasNext:  hasNext,
			},
		},
	}
}

// NewPageResponseWithExtra creates a new paginated success response with extra data
func NewPageResponseWithExtra[T any](message string, list []T, current, pageSize, total int64, extra any) Response[PageData[T]] {
	hasNext := current*pageSize < total

	return Response[PageData[T]]{
		Status:  StatusSuccess,
		Message: message,
		Data: PageData[T]{
			List: list,
			Pagination: Pagination{
				Current:  current,
				PageSize: pageSize,
				Total:    total,
				HasNext:  hasNext,
			},
			Extra: extra,
		},
	}
}

// NewErrorResponse creates a new error response.
// If err is an *AppError, it extracts the code and user-safe message automatically.
// Otherwise, it falls back to a generic "INTERNAL_ERROR" code.
func NewErrorResponse(message string, err error) Response[any] {
	errorDetails := &ErrorDetails{
		Code:    "INTERNAL_ERROR",
		Message: message,
	}

	var appErr *domainerrors.AppError
	if errors.As(err, &appErr) {
		errorDetails.Code = appErr.Code
		errorDetails.Message = appErr.Message
	} else if err != nil {
		// For non-AppError errors, use the caller-provided message.
		// The raw err.Error() is intentionally NOT exposed to prevent
		// leaking internal details (e.g. SQL errors) to the client.
		errorDetails.Message = message
	}

	return Response[any]{
		Status:  StatusError,
		Message: message,
		Error:   errorDetails,
	}
}

// HTTPCodeFromError extracts the HTTP status code from an error.
// If the error is an *AppError, it returns the AppError's HTTPCode.
// Otherwise, it returns the provided fallback status code.
func HTTPCodeFromError(err error, fallback int) int {
	var appErr *domainerrors.AppError
	if errors.As(err, &appErr) {
		return appErr.HTTPCode
	}
	return fallback
}

// JSON converts the response to JSON bytes
func (r Response[T]) JSON() ([]byte, error) {
	return json.Marshal(r)
}

