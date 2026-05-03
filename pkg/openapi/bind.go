package openapi

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/gin-gonic/gin"
)

// bindInput populates an input struct from the Gin request.
//
// Field binding rules (by struct tag):
//
//	Body          → c.ShouldBindJSON (triggers binding:"required" validation)
//	path:"name"   → c.Param + type conversion
//	query:"name"  → c.Query + type conversion
//	header:"name" → c.GetHeader
//
// If the input type is Empty (no fields), binding is skipped entirely.
func bindInput(c *gin.Context, inputPtr any) error {
	v := reflect.ValueOf(inputPtr).Elem()
	t := v.Type()

	if t == reflect.TypeFor[Empty]() {
		return nil
	}

	for i := range t.NumField() {
		field := t.Field(i)
		fv := v.Field(i)

		// Body field — bind JSON
		if field.Name == "Body" {
			bodyPtr := reflect.New(field.Type)
			if err := c.ShouldBindJSON(bodyPtr.Interface()); err != nil {
				return fmt.Errorf("invalid request body: %w", err)
			}
			fv.Set(bodyPtr.Elem())
			continue
		}

		// Path parameter
		if tag := field.Tag.Get("path"); tag != "" {
			raw := c.Param(tag)
			if raw == "" {
				continue
			}
			if err := setFieldFromString(fv, raw); err != nil {
				return fmt.Errorf("invalid path param %q: %w", tag, err)
			}
			continue
		}

		// Query parameter
		if tag := field.Tag.Get("query"); tag != "" {
			raw := c.Query(tag)
			if raw == "" {
				continue
			}
			if err := setFieldFromString(fv, raw); err != nil {
				return fmt.Errorf("invalid query param %q: %w", tag, err)
			}
			continue
		}

		// Header
		if tag := field.Tag.Get("header"); tag != "" {
			raw := c.GetHeader(tag)
			if raw == "" {
				continue
			}
			if err := setFieldFromString(fv, raw); err != nil {
				return fmt.Errorf("invalid header %q: %w", tag, err)
			}
			continue
		}
	}

	return nil
}

// setFieldFromString converts a string value to the target field type.
func setFieldFromString(fv reflect.Value, raw string) error {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return err
		}
		fv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		fv.SetFloat(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		fv.SetBool(b)
	default:
		return fmt.Errorf("unsupported field type: %s", fv.Type())
	}
	return nil
}
