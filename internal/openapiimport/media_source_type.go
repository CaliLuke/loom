package openapiimport

import (
	"github.com/pb33f/libopenapi/datamodel/high/base"

	"github.com/CaliLuke/loom/internal/openapiref"
)

// sourceEncodingType retains the source's effective non-null type before value
// normalization. Reference proxies resolve to their source; allOf contributes
// type assertions. Alternative combinators never manufacture a missing type.
func sourceEncodingType(source *base.Schema, active map[*base.Schema]bool) string {
	if source == nil || active[source] {
		return "absent"
	}
	active[source] = true
	defer delete(active, source)
	result := "absent"
	for _, value := range source.Type {
		if value == "null" {
			continue
		}
		if result != "absent" && result != value {
			return "absent"
		}
		result = value
	}
	if result != "absent" {
		return result
	}
	for _, branch := range source.AllOf {
		if branch != nil {
			if value := sourceEncodingType(branch.Schema(), active); value != "absent" {
				result = value
			}
		}
	}
	return result
}

// schemaEncodingType returns only source facts lost by native type inference.
// This keeps ordinary concrete schemas compact while preserving typed unions
// and typeless singleton/nullable compositions after normalization.
func schemaEncodingType(schema *Schema, schemas map[string]NamedSchema) string {
	if schema == nil || schema.sourceType == "" || schema.sourceType == schema.Type ||
		schema.sourceType == "absent" && schema.Type == "" && schema.Ref == "" {
		return ""
	}
	if schema.Ref != "" && inheritedEncodingType(schema.Ref, schemas) == schema.sourceType {
		return ""
	}
	return schema.sourceType
}

// inheritedEncodingType compares source facts at alias boundaries, before an
// occurrence adds metadata that would unnecessarily create a reference overlay.
func inheritedEncodingType(ref string, schemas map[string]NamedSchema) string {
	seen := make(map[string]bool)
	for ref != "" && !seen[ref] {
		seen[ref] = true
		name, err := openapiref.Name(ref, "#/components/schemas/")
		if err != nil {
			return ""
		}
		schema := schemas[name].Schema
		if schema == nil {
			return ""
		}
		if schema.sourceType != "" {
			return schema.sourceType
		}
		if schema.Ref == "" {
			if schema.Type != "" {
				return schema.Type
			}
			return "absent"
		}
		ref = schema.Ref
	}
	return ""
}
