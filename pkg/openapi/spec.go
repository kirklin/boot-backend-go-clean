package openapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// Empty can be used as a type parameter when there is no request body or
// no response body.
type Empty struct{}

type serverEntry struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type securitySchemeEntry struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
	Name         string `json:"name,omitempty"`
	In           string `json:"in,omitempty"`
}

// registeredOp stores an operation together with its reflected types.
type registeredOp struct {
	op       operation
	reqType  reflect.Type
	respType reflect.Type
}

// ---------------------------------------------------------------------------
// Spec
// ---------------------------------------------------------------------------

// Spec collects OpenAPI metadata and generates the final 3.1 document.
type Spec struct {
	title           string
	version         string
	description     string
	servers         []serverEntry
	securitySchemes map[string]securitySchemeEntry
	ops             []registeredOp
	responseWrapper reflect.Type // optional: wraps every response in this type
}

// NewSpec creates a new OpenAPI 3.1 spec builder.
func NewSpec(title, version string) *Spec {
	return &Spec{
		title:           title,
		version:         version,
		securitySchemes: make(map[string]securitySchemeEntry),
	}
}

// SetDescription sets the API-level description.
func (s *Spec) SetDescription(d string) *Spec {
	s.description = d
	return s
}

// AddServer adds an API server URL (e.g. "/v1/api").
func (s *Spec) AddServer(url, description string) *Spec {
	s.servers = append(s.servers, serverEntry{URL: url, Description: description})
	return s
}

// AddBearerAuth registers a Bearer JWT security scheme.
func (s *Spec) AddBearerAuth(name string) *Spec {
	s.securitySchemes[name] = securitySchemeEntry{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
	}
	return s
}

// AddAPIKeyAuth registers an API key security scheme.
func (s *Spec) AddAPIKeyAuth(name, header string) *Spec {
	s.securitySchemes[name] = securitySchemeEntry{
		Type: "apiKey",
		Name: header,
		In:   "header",
	}
	return s
}

// addOperation records a route for spec generation (called by api.go).
func (s *Spec) addOperation(op operation, reqType, respType reflect.Type) {
	s.ops = append(s.ops, registeredOp{
		op:       op,
		reqType:  reqType,
		respType: respType,
	})
}

// Build generates the complete OpenAPI 3.1 JSON document.
func (s *Spec) Build() ([]byte, error) {
	sg := newSchemaGenerator()

	// Build paths
	paths := make(map[string]map[string]any)

	for _, reg := range s.ops {
		op := reg.op

		// Initialize path item if needed
		if _, ok := paths[op.path]; !ok {
			paths[op.path] = make(map[string]any)
		}

		opObj := make(map[string]any)

		if op.id != "" {
			opObj["operationId"] = op.id
		}
		if op.summary != "" {
			opObj["summary"] = op.summary
		}
		if op.description != "" {
			opObj["description"] = op.description
		}
		if len(op.tags) > 0 {
			opObj["tags"] = op.tags
		}
		if op.deprecated {
			opObj["deprecated"] = true
		}

		// Path parameters
		pathParams := extractPathParams(op.path)
		if len(pathParams) > 0 {
			var params []map[string]any
			for _, name := range pathParams {
				params = append(params, map[string]any{
					"name":     name,
					"in":       "path",
					"required": true,
					"schema":   map[string]any{"type": "string"},
				})
			}
			opObj["parameters"] = params
		}

		// Request body
		reqRef := sg.schemaRef(reg.reqType)
		if reqRef != nil {
			opObj["requestBody"] = map[string]any{
				"required": true,
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": reqRef,
					},
				},
			}
		}

		// Response
		respRef := sg.schemaRef(reg.respType)
		statusCode := op.status
		if statusCode == 0 {
			statusCode = 200
		}
		statusStr := http.StatusText(statusCode)
		if statusStr == "" {
			statusStr = "Success"
		}

		responseSchema := s.buildResponseSchema(respRef)
		responses := map[string]any{
			statusCodeStr(statusCode): map[string]any{
				"description": statusStr,
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": responseSchema,
					},
				},
			},
		}
		opObj["responses"] = responses

		// Security
		if len(op.security) > 0 {
			var sec []map[string][]string
			for _, name := range op.security {
				sec = append(sec, map[string][]string{name: {}})
			}
			opObj["security"] = sec
		}

		paths[op.path][strings.ToLower(op.method)] = opObj
	}

	// Build the top-level document
	doc := map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":   s.title,
			"version": s.version,
		},
	}

	if s.description != "" {
		doc["info"].(map[string]any)["description"] = s.description
	}

	if len(s.servers) > 0 {
		doc["servers"] = s.servers
	}

	// Sort paths for deterministic output
	sortedPaths := make(map[string]any)
	pathKeys := make([]string, 0, len(paths))
	for k := range paths {
		pathKeys = append(pathKeys, k)
	}
	sort.Strings(pathKeys)
	for _, k := range pathKeys {
		sortedPaths[k] = paths[k]
	}
	doc["paths"] = sortedPaths

	// Components
	components := make(map[string]any)

	// Schemas from reflected types
	defs := sg.defsToComponentSchemas()
	if len(defs) > 0 {
		components["schemas"] = defs
	}

	// Security schemes
	if len(s.securitySchemes) > 0 {
		components["securitySchemes"] = s.securitySchemes
	}

	if len(components) > 0 {
		doc["components"] = components
	}

	return json.MarshalIndent(doc, "", "  ")
}

// Mount registers the OpenAPI spec endpoint and docs UI on the Gin engine.
//
//	GET /openapi.json → the generated spec
//	GET /docs         → Scalar interactive docs
func (s *Spec) Mount(engine *gin.Engine) {
	specJSON, err := s.Build()
	if err != nil {
		panic("openapi: failed to build spec: " + err.Error())
	}

	engine.GET("/openapi.json", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json; charset=utf-8", specJSON)
	})

	engine.GET("/docs", func(c *gin.Context) {
		html := renderScalarHTML(s.title, "/openapi.json")
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	})
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// buildResponseSchema wraps the response data type in the project's standard
// response envelope: { status, message, data, error }.
func (s *Spec) buildResponseSchema(dataSchema map[string]any) map[string]any {
	envelope := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"status": map[string]any{
				"type":        "string",
				"description": "Response status (success or error)",
				"example":     "success",
			},
			"message": map[string]any{
				"type":        "string",
				"description": "Human-readable response message",
			},
			"timestamp": map[string]any{
				"type":        "string",
				"format":      "date-time",
				"description": "ISO 8601 timestamp",
			},
		},
		"required": []string{"status", "message", "timestamp"},
	}

	props := envelope["properties"].(map[string]any)

	if dataSchema != nil {
		props["data"] = dataSchema
	}

	return envelope
}

func statusCodeStr(code int) string {
	return fmt.Sprintf("%d", code)
}
