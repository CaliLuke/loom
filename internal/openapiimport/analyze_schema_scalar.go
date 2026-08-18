package openapiimport

import (
	"fmt"

	"github.com/pb33f/libopenapi/datamodel/high/base"
)

// schemaDefault decodes the JSON Schema default keyword and retains it when
// the schema is one of the scalar types Loom's Default DSL can express and
// the decoded value matches that type. Composite-type defaults (object,
// array) and type-mismatched defaults remain in the strict import subset's
// generic "schema-keyword" diagnostic rather than risk emitting a design that
// fails to evaluate.
func (a *analyzer) schemaDefault(schema *Schema, source *base.Schema, path string) {
	if source.Default == nil {
		return
	}
	var decoded any
	if err := source.Default.Decode(&decoded); err != nil {
		a.unsupported("schema-keyword", path, fmt.Sprintf("default value cannot be decoded: %v", err))
		return
	}
	if !defaultCompatibleWithType(schema.Type, decoded) {
		a.unsupported("schema-keyword", path, fmt.Sprintf("default value is not compatible with the strict import subset for schema type %q", schema.Type))
		return
	}
	schema.Default = &SchemaDefault{Value: decoded}
}

// defaultCompatibleWithType reports whether a decoded default value can be
// rendered as a Loom Default(...) call for the given normalized schema type.
// Only scalar types are supported; object and array defaults, and an
// explicit null default on any type, are left unsupported.
func defaultCompatibleWithType(schemaType string, value any) bool {
	if value == nil {
		return false
	}
	switch schemaType {
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "integer":
		switch value.(type) {
		case int, int64, uint64:
			return true
		}
		return false
	case "number":
		switch value.(type) {
		case int, int64, uint64, float32, float64:
			return true
		}
		return false
	default:
		return false
	}
}

// schemaFormat reports an unrecognized string, integer, or number format as a
// lossy-allowed diagnostic. An empty format is treated as absent (OpenAPI 3.1
// permits arbitrary format values, and JSON Schema requires that unknown
// formats never fail validation or processing), so it never reports here;
// render_schema.go falls back to the type's widest unformatted representation
// for anything not recognized.
func (a *analyzer) schemaFormat(schema *Schema, path string) {
	if schema.Format == "" {
		return
	}
	switch schema.Type {
	case "string":
		if schema.Format == "byte" || schema.Format == "binary" {
			return
		}
		if _, ok := stringFormatDSL(schema.Format); !ok {
			a.unsupported("schema-format", path, fmt.Sprintf("string format %q is not a recognized Loom format; rendering without a format validation", schema.Format))
		}
	case "integer":
		if schema.Format != "int32" && schema.Format != "int64" {
			a.unsupported("schema-format", path, fmt.Sprintf("integer format %q is not a recognized Loom format; rendering as an unformatted integer", schema.Format))
		}
	case "number":
		if schema.Format != "float" && schema.Format != "double" {
			a.unsupported("schema-format", path, fmt.Sprintf("number format %q is not a recognized Loom format; rendering as Float64", schema.Format))
		}
	}
}
