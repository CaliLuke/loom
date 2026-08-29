package openapi

import (
	"github.com/CaliLuke/loom/expr"
)

// MustGenerate returns true if the meta indicates that a OpenAPI specification should be
// generated, false otherwise.
func MustGenerate(meta expr.MetaExpr) bool {
	m, ok := meta.Last("openapi:generate")
	if ok && m == "false" {
		return false
	}
	return true
}

// AdditionalPropertiesFromExpr extracts the OpenAPI additionalProperties.
func AdditionalPropertiesFromExpr(meta expr.MetaExpr) any {
	m, ok := meta.Last("openapi:additionalProperties")
	if ok && m == "false" {
		return false
	}
	return nil
}

// ClosedObjectModeFromExpr reports whether OpenAPI closed object mode is
// enabled via metadata.
func ClosedObjectModeFromExpr(meta expr.MetaExpr) bool {
	m, ok := meta.Last("openapi:closed-objects")
	return ok && m == "true"
}
