package middleware

// Gin context keys used across middleware to share request-scoped data.
// Centralizing these prevents typo-induced bugs when keys are used in
// multiple middleware and controller files.
const (
	// ContextKeyRequestID is the gin context key for the unique request identifier.
	// Set by RequestIDMiddleware, consumed by AccessLogMiddleware and others.
	ContextKeyRequestID = "x-request-id"

	// ContextKeyUserID is the gin context key for the authenticated user's ID.
	// Set by JWTAuthMiddleware after token validation.
	ContextKeyUserID = "x-user-id"

	// ContextKeyUsername is the gin context key for the authenticated user's username.
	// Set by JWTAuthMiddleware after token validation.
	ContextKeyUsername = "x-username"

	// HeaderRequestID is the HTTP header name for request tracing.
	HeaderRequestID = "X-Request-ID"
)
