package openapi

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"

	domainerrors "github.com/kirklin/boot-backend-go-clean/internal/domain/errors"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/entity/response"
)

// ---------------------------------------------------------------------------
// API — thin wrapper around gin.RouterGroup
// ---------------------------------------------------------------------------

// API wraps a gin.RouterGroup and a shared Spec. Every route registered
// through the API methods is also recorded in the Spec for OpenAPI output.
type API struct {
	group *gin.RouterGroup
	spec  *Spec
}

// NewAPI creates a new API that wraps the given RouterGroup.
// All operations registered through this API are collected in spec.
func NewAPI(group *gin.RouterGroup, spec *Spec) *API {
	return &API{group: group, spec: spec}
}

// Group creates a sub-API with a path prefix, mirroring gin.RouterGroup.Group.
// Middleware attached here applies to all routes in the sub-group.
func (a *API) Group(path string, mw ...gin.HandlerFunc) *API {
	return &API{group: a.group.Group(path, mw...), spec: a.spec}
}

// Use attaches middleware to the current group.
func (a *API) Use(mw ...gin.HandlerFunc) {
	a.group.Use(mw...)
}

// ---------------------------------------------------------------------------
// Handler — the typed handler signature
// ---------------------------------------------------------------------------

// Handler is a type-safe handler function.
// I = input struct (bound from path/query/header/body)
// O = output struct (the data portion of the response)
type Handler[I, O any] func(ctx context.Context, input *I) (*O, error)

// ---------------------------------------------------------------------------
// Generic registration — the core of the package
// ---------------------------------------------------------------------------

// Post registers a POST route with a typed handler + OpenAPI metadata.
func Post[I, O any](api *API, path string, h Handler[I, O], opts ...Option) {
	register[I, O](api, "POST", path, h, opts)
}

// Get registers a GET route.
func Get[I, O any](api *API, path string, h Handler[I, O], opts ...Option) {
	register[I, O](api, "GET", path, h, opts)
}

// Put registers a PUT route.
func Put[I, O any](api *API, path string, h Handler[I, O], opts ...Option) {
	register[I, O](api, "PUT", path, h, opts)
}

// Delete registers a DELETE route.
func Delete[I, O any](api *API, path string, h Handler[I, O], opts ...Option) {
	register[I, O](api, "DELETE", path, h, opts)
}

// Patch registers a PATCH route.
func Patch[I, O any](api *API, path string, h Handler[I, O], opts ...Option) {
	register[I, O](api, "PATCH", path, h, opts)
}

// ---------------------------------------------------------------------------
// Raw registration — for routes that need raw gin.HandlerFunc
// (e.g. infrastructure endpoints, Prometheus, pprof)
// ---------------------------------------------------------------------------

// RawGet registers a GET route with a raw gin.HandlerFunc + OpenAPI metadata.
// The handler is NOT wrapped — no auto-binding or response wrapping occurs.
func RawGet(api *API, path string, h gin.HandlerFunc, opts ...Option) {
	rawRegister(api, "GET", path, h, opts)
}

// RawPost registers a POST route with a raw gin.HandlerFunc.
func RawPost(api *API, path string, h gin.HandlerFunc, opts ...Option) {
	rawRegister(api, "POST", path, h, opts)
}

func rawRegister(api *API, method, path string, h gin.HandlerFunc, opts []Option) {
	op := applyOptions(method, opts)

	fullPath := joinPath(api.group.BasePath(), path)
	op.path = ginPathToOpenAPI(fullPath)

	handlers := append(op.middlewares, h)
	switch method {
	case "GET":
		api.group.GET(path, handlers...)
	case "POST":
		api.group.POST(path, handlers...)
	case "PUT":
		api.group.PUT(path, handlers...)
	case "DELETE":
		api.group.DELETE(path, handlers...)
	case "PATCH":
		api.group.PATCH(path, handlers...)
	}

	// Record in spec with nil types (no schema generation needed)
	api.spec.addOperation(op, nil, nil)
}

// register is the common implementation for all HTTP method helpers.
func register[I, O any](api *API, method, path string, h Handler[I, O], opts []Option) {
	op := applyOptions(method, opts)

	// Build the full Gin path for the spec (group prefix + local path).
	fullPath := joinPath(api.group.BasePath(), path)
	op.path = ginPathToOpenAPI(fullPath)

	// Create the gin.HandlerFunc that wraps the typed handler.
	ginHandler := wrapHandler[I, O](h, op)

	// Register with Gin — per-op middleware first, then handler.
	handlers := append(op.middlewares, ginHandler)
	switch method {
	case "GET":
		api.group.GET(path, handlers...)
	case "POST":
		api.group.POST(path, handlers...)
	case "PUT":
		api.group.PUT(path, handlers...)
	case "DELETE":
		api.group.DELETE(path, handlers...)
	case "PATCH":
		api.group.PATCH(path, handlers...)
	}

	// Determine request type for spec — use Body field type if present.
	reqType := getRequestSchemaType[I]()
	respType := reflect.TypeFor[O]()

	// Collect the operation for OpenAPI spec generation.
	api.spec.addOperation(op, reqType, respType)
}

// wrapHandler creates a gin.HandlerFunc from a typed Handler.
// It handles: binding → context propagation → handler call → error mapping → response.
func wrapHandler[I, O any](h Handler[I, O], op operation) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Bind input from request
		var input I
		if err := bindInput(c, &input); err != nil {
			writeError(c, domainerrors.ErrBadRequest.Wrap(err))
			return
		}

		// 2. Propagate gin context values to context.Context
		ctx := withGinContext(c)

		// 3. Call the typed handler
		output, err := h(ctx, &input)

		// 4. Handle error → automatic AppError → HTTP status mapping
		if err != nil {
			writeError(c, err)
			return
		}

		// 5. Write success response (auto-wrapped in response.Response[T])
		status := op.status
		if status == 0 {
			status = http.StatusOK
		}

		msg := op.message
		if msg == "" {
			msg = http.StatusText(status)
		}

		if output == nil {
			c.JSON(status, response.NewSuccessResponse[any](msg, nil))
			return
		}
		c.JSON(status, response.NewSuccessResponse(msg, output))
	}
}

// writeError maps an error to the appropriate HTTP response.
// If the error is an AppError, it uses the AppError's HTTPCode and Code.
// Otherwise, it returns a generic 500.
func writeError(c *gin.Context, err error) {
	var appErr *domainerrors.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.HTTPCode, response.NewErrorResponse(appErr.Message, err))
		return
	}
	c.JSON(http.StatusInternalServerError,
		response.NewErrorResponse("Internal server error", err))
}

// getRequestSchemaType returns the type to use for request body schema.
// If I has a Body field, return the Body field's type (that's what goes in the request body).
// If I is Empty or only has path/query/header tags, return nil (no request body).
func getRequestSchemaType[I any]() reflect.Type {
	t := reflect.TypeFor[I]()
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Empty → no request body
	if t == reflect.TypeFor[Empty]() {
		return nil
	}

	// Has a Body field → use its type
	if bodyField, ok := t.FieldByName("Body"); ok {
		return bodyField.Type
	}

	// Check if ALL fields are path/query/header params → no request body
	hasOnlyParamFields := true
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Tag.Get("path") == "" && f.Tag.Get("query") == "" && f.Tag.Get("header") == "" {
			hasOnlyParamFields = false
			break
		}
	}
	if hasOnlyParamFields {
		return nil
	}

	// Fallback: use the whole type as request body
	return t
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// joinPath joins two URL path segments, avoiding double slashes.
func joinPath(base, path string) string {
	if base == "" || base == "/" {
		return path
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
}

// ginPathToOpenAPI converts Gin :param to OpenAPI {param} syntax.
func ginPathToOpenAPI(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, ":") {
			parts[i] = "{" + p[1:] + "}"
		} else if strings.HasPrefix(p, "*") {
			parts[i] = "{" + p[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}

// extractPathParams returns parameter names from an OpenAPI-style path.
func extractPathParams(path string) []string {
	var params []string
	parts := strings.Split(path, "/")
	for _, p := range parts {
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			params = append(params, p[1:len(p)-1])
		}
	}
	return params
}
