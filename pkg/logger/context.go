package logger

import "context"

type ctxKey struct{}

// NewContext returns a copy of the parent context with the given Logger attached.
// Use this to propagate a request-scoped logger (with RequestID, UserID, etc.)
// through the call chain.
func NewContext(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext extracts the Logger from the context.
// If no logger is found, it returns the global logger as a safe fallback.
func FromContext(ctx context.Context) Logger {
	if l, ok := ctx.Value(ctxKey{}).(Logger); ok {
		return l
	}
	return GetLogger()
}
