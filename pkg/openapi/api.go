package openapi

import (
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
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
// Generic registration — the core of the package
// ---------------------------------------------------------------------------

// Post registers a POST route. Req and Resp are used only for OpenAPI
// schema reflection — they do NOT affect the runtime handler in any way.
func Post[Req, Resp any](api *API, path string, handler gin.HandlerFunc, opts ...Option) {
	register[Req, Resp](api, "POST", path, handler, opts)
}

// Get registers a GET route.
func Get[Req, Resp any](api *API, path string, handler gin.HandlerFunc, opts ...Option) {
	register[Req, Resp](api, "GET", path, handler, opts)
}

// Put registers a PUT route.
func Put[Req, Resp any](api *API, path string, handler gin.HandlerFunc, opts ...Option) {
	register[Req, Resp](api, "PUT", path, handler, opts)
}

// Delete registers a DELETE route.
func Delete[Req, Resp any](api *API, path string, handler gin.HandlerFunc, opts ...Option) {
	register[Req, Resp](api, "DELETE", path, handler, opts)
}

// Patch registers a PATCH route.
func Patch[Req, Resp any](api *API, path string, handler gin.HandlerFunc, opts ...Option) {
	register[Req, Resp](api, "PATCH", path, handler, opts)
}

// register is the common implementation for all HTTP method helpers.
func register[Req, Resp any](api *API, method, path string, handler gin.HandlerFunc, opts []Option) {
	op := applyOptions(method, opts)

	// Build the full Gin path for the spec (group prefix + local path).
	fullPath := joinPath(api.group.BasePath(), path)
	op.path = ginPathToOpenAPI(fullPath)

	// Register with Gin — middleware first, then handler.
	handlers := append(op.middlewares, handler)
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

	// Collect the operation for OpenAPI spec generation.
	api.spec.addOperation(op, reflect.TypeFor[Req](), reflect.TypeFor[Resp]())
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
