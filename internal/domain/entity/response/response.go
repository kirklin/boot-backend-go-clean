package response

import "encoding/json"

// ResponseStatus represents the status of the response
type ResponseStatus string

const (
	StatusSuccess ResponseStatus = "success"
	StatusError   ResponseStatus = "error"
)

// Response is a generic structure for all API responses
type Response[T any] struct {
	Status  ResponseStatus `json:"status"`
	Message string         `json:"message"`
	Data    T              `json:"data,omitempty"`
	Error   *ErrorDetails  `json:"error,omitempty"`
}

// ErrorDetails contains detailed error information
type ErrorDetails struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

// NewSuccessResponse creates a new success response
func NewSuccessResponse[T any](message string, data T) Response[T] {
	return Response[T]{
		Status:  StatusSuccess,
		Message: message,
		Data:    data,
	}
}

// NewErrorResponse creates a new error response
func NewErrorResponse(message string, err error) Response[any] {
	errorDetails := &ErrorDetails{
		Message: message,
	}

	if err != nil {
		errorDetails.Code = "INTERNAL_ERROR"
		errorDetails.Message = err.Error()
	}

	return Response[any]{
		Status:  StatusError,
		Message: message,
		Error:   errorDetails,
	}
}

// JSON converts the response to JSON bytes
func (r Response[T]) JSON() ([]byte, error) {
	return json.Marshal(r)
}
