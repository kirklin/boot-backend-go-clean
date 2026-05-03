// Package openapi provides a typed API framework on top of Gin that
// simultaneously handles request binding, response wrapping, error mapping,
// and OpenAPI 3.1 spec generation.
//
// Controller handlers use the signature:
//
//	func(ctx context.Context, input *I) (*O, error)
//
// The framework automatically binds I from path/query/header/body,
// wraps O in the standard response.Response[T] envelope,
// maps errors via AppError → HTTP status, and collects OpenAPI metadata.
package openapi

import "github.com/gin-gonic/gin"

// ---------------------------------------------------------------------------
// Operation metadata (internal)
// ---------------------------------------------------------------------------

type operation struct {
	method      string
	path        string // full path including group prefix
	id          string
	summary     string
	description string
	tags        []string
	status      int    // success status code, default 200
	message     string // success response message
	deprecated  bool
	security    []string          // references to named security schemes
	middlewares []gin.HandlerFunc // per-operation middleware
}

// ---------------------------------------------------------------------------
// Functional options
// ---------------------------------------------------------------------------

// Option configures an individual route's OpenAPI metadata.
type Option func(*operation)

// ID sets the operationId.
func ID(id string) Option { return func(o *operation) { o.id = id } }

// Summary sets a short description for the operation.
func Summary(s string) Option { return func(o *operation) { o.summary = s } }

// Description sets a longer description for the operation.
func Description(d string) Option { return func(o *operation) { o.description = d } }

// Tags groups the operation under one or more tags.
func Tags(t ...string) Option { return func(o *operation) { o.tags = t } }

// Status overrides the default success HTTP status code (200).
func Status(code int) Option { return func(o *operation) { o.status = code } }

// Security references a named security scheme registered via Spec.AddBearerAuth.
func Security(names ...string) Option { return func(o *operation) { o.security = names } }

// Message sets the success response message (e.g. "Login successful").
func Message(m string) Option { return func(o *operation) { o.message = m } }

// Deprecated marks the operation as deprecated in the spec.
func Deprecated() Option { return func(o *operation) { o.deprecated = true } }

// Middleware attaches per-operation Gin middleware (e.g. JWT guard for a
// single route). These run before the handler.
func Middleware(mw ...gin.HandlerFunc) Option {
	return func(o *operation) { o.middlewares = append(o.middlewares, mw...) }
}

func applyOptions(method string, opts []Option) operation {
	op := operation{method: method, status: 200}
	for _, fn := range opts {
		fn(&op)
	}
	return op
}
