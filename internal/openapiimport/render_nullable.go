package openapiimport

import (
	"strings"
)

func (r *renderer) effectiveUnconstrained(schema *Schema) bool {
	if schema == nil {
		return false
	}
	if schema.Unconstrained {
		return true
	}
	if schema.Ref == "" {
		return false
	}
	name := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
	named, ok := r.schemas[name]
	return ok && named.Schema != nil && named.Schema.Unconstrained
}

func (r *renderer) effectiveNullable(schema *Schema) bool {
	if schema == nil {
		return false
	}
	if schema.Nullable {
		return true
	}
	if schema.Ref == "" {
		return false
	}
	name := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
	named, ok := r.schemas[name]
	return ok && named.Schema != nil && named.Schema.Nullable
}

func (r *renderer) emitAttributeGoType(schema *Schema, path string) error {
	if r.effectiveUnconstrained(schema) {
		return nil
	}
	return r.emitNullableGoType(schema, path)
}
