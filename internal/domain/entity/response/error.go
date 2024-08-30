package response

type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func NewErrorResponse(message string, err error) ErrorResponse {
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}

	return ErrorResponse{
		Status:  "error",
		Message: message,
		Error:   errorMessage,
	}
}
