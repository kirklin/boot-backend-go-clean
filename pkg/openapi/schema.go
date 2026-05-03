package openapi

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/invopop/jsonschema"
)

// schemaGenerator handles Go type → JSON Schema conversion.
type schemaGenerator struct {
	reflector *jsonschema.Reflector
	defs      map[string]*jsonschema.Schema // collected definitions
}

func newSchemaGenerator() *schemaGenerator {
	r := &jsonschema.Reflector{
		DoNotReference:            false,
		Anonymous:                 true, // don't add package-based $id
		AllowAdditionalProperties: true,
	}
	return &schemaGenerator{
		reflector: r,
		defs:      make(map[string]*jsonschema.Schema),
	}
}

// schemaRef generates the JSON Schema for the given type, stores any
// definitions, and returns a map that can be used as a JSON Schema
// reference (either {"$ref": "..."} or an inline schema for primitives).
func (sg *schemaGenerator) schemaRef(t reflect.Type) map[string]any {
	// Unwrap pointer types.
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Handle special cases.
	if t == reflect.TypeFor[Empty]() {
		return nil
	}

	schema := sg.reflector.ReflectFromType(t)

	// Collect all definitions ($defs is a plain map[string]*Schema).
	for name, def := range schema.Definitions {
		sg.defs[name] = def
	}

	// If the schema is a $ref, convert to components/schemas reference.
	if schema.Ref != "" {
		refName := strings.TrimPrefix(schema.Ref, "#/$defs/")
		return map[string]any{"$ref": "#/components/schemas/" + refName}
	}

	// For simple types, return an inline schema.
	return sg.schemaToMap(schema)
}

// schemaToMap converts a jsonschema.Schema to a plain map for JSON marshaling.
func (sg *schemaGenerator) schemaToMap(s *jsonschema.Schema) map[string]any {
	// Easiest approach: marshal to JSON, then unmarshal to map.
	// This handles all the complex nested fields correctly.
	data, err := json.Marshal(s)
	if err != nil {
		return map[string]any{"type": "object"}
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]any{"type": "object"}
	}

	// Rewrite any $defs references to components/schemas references.
	sg.rewriteRefs(m)

	// Remove $schema and $id from nested schemas (only needed at top level).
	delete(m, "$schema")
	delete(m, "$id")

	// Move $defs into our collected defs and remove from the inline schema.
	if defs, ok := m["$defs"].(map[string]any); ok {
		for name, def := range defs {
			if defMap, ok := def.(map[string]any); ok {
				// Store as raw map — we'll output it directly.
				defBytes, _ := json.Marshal(defMap)
				var defSchema jsonschema.Schema
				if err := json.Unmarshal(defBytes, &defSchema); err == nil {
					sg.defs[name] = &defSchema
				}
			}
		}
		delete(m, "$defs")
	}

	return m
}

// rewriteRefs recursively rewrites "#/$defs/Foo" → "#/components/schemas/Foo".
func (sg *schemaGenerator) rewriteRefs(m map[string]any) {
	for k, v := range m {
		if k == "$ref" {
			if s, ok := v.(string); ok && strings.HasPrefix(s, "#/$defs/") {
				m[k] = "#/components/schemas/" + strings.TrimPrefix(s, "#/$defs/")
			}
		}
		if sub, ok := v.(map[string]any); ok {
			sg.rewriteRefs(sub)
		}
		if arr, ok := v.([]any); ok {
			for _, item := range arr {
				if sub, ok := item.(map[string]any); ok {
					sg.rewriteRefs(sub)
				}
			}
		}
	}
}

// defsToComponentSchemas returns all collected definitions as a map
// suitable for the "components.schemas" section of the OpenAPI spec.
func (sg *schemaGenerator) defsToComponentSchemas() map[string]any {
	result := make(map[string]any)
	for name, schema := range sg.defs {
		m := sg.schemaToMap(schema)
		delete(m, "$defs") // clean up any leftover nested defs
		result[name] = m
	}
	return result
}
