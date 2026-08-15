package openapiimport

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

func (r *renderer) emitNullableJSONTag(name string, schema *Schema) {
	if r.effectiveNullable(schema) || r.effectiveUnconstrained(schema) {
		r.line("Meta(%q, %q)", "struct:tag:json:name", name+",omitzero")
	}
}

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
		r.line(
			"Meta(%q, %q, %q, %q)",
			"struct:field:type",
			"loom.Nullable[any]",
			"github.com/CaliLuke/loom/pkg",
			"loom",
		)
		return nil
	}
	return r.emitNullableGoType(schema, path)
}

func nullablePrimitiveGoType(schema *Schema) (string, bool) {
	switch schema.Type {
	case "string":
		if schema.Format == "byte" || schema.Format == "binary" {
			return "[]byte", true
		}
		return "string", true
	case "integer":
		switch schema.Format {
		case "int32":
			return "int32", true
		case "int64":
			return "int64", true
		default:
			return "int", true
		}
	case "number":
		if schema.Format == "float" {
			return "float32", true
		}
		return "float64", true
	case "boolean":
		return "bool", true
	default:
		return "", false
	}
}

func (r *renderer) nullableArrayGoType(schema *Schema, path string) (string, bool, error) {
	itemType, itemObject, err := r.schemaGoType(schema.Items, path+"/items")
	if err != nil {
		return "", false, err
	}
	if r.effectiveNullable(schema.Items) {
		itemType = "loom.Nullable[" + itemType + "]"
	} else if itemObject && !strings.HasPrefix(itemType, "*") {
		itemType = "*" + itemType
	}
	return "[]" + itemType, false, nil
}

func (r *renderer) nullableObjectGoType(schema *Schema, path string) (string, bool, error) {
	if schema.AdditionalProperties == nil || schema.AdditionalProperties.Schema == nil {
		return r.inlineObjectGoType(schema, path)
	}
	valueSchema := schema.AdditionalProperties.Schema
	valueType, valueObject, err := r.schemaGoType(valueSchema, path+"/additionalProperties")
	if err != nil {
		return "", false, err
	}
	if r.effectiveNullable(valueSchema) {
		valueType = "loom.Nullable[" + valueType + "]"
	} else if valueObject && !strings.HasPrefix(valueType, "*") {
		valueType = "*" + valueType
	}
	return "map[string]" + valueType, false, nil
}

func (r *renderer) inlineObjectGoType(schema *Schema, path string) (string, bool, error) {
	required := make(map[string]struct{}, len(schema.Required))
	for _, name := range schema.Required {
		required[name] = struct{}{}
	}
	used := make(map[string]int)
	fields := make([]string, 0, len(schema.Properties))
	for _, property := range schema.Properties {
		fieldName := uniqueName(codegen.Goify(property.Name, true), used)
		fieldType, object, err := r.schemaGoType(property.Schema, path+"/properties/"+escapeJSONPointer(property.Name))
		if err != nil {
			return "", false, err
		}
		_, isRequired := required[property.Name]
		switch {
		case r.effectiveUnconstrained(property.Schema):
			fieldType = "loom.Nullable[any]"
		case r.effectiveNullable(property.Schema):
			fieldType = "loom.Nullable[" + fieldType + "]"
		case object || !isRequired && property.Schema.Type != "array" && property.Schema.Type != "object":
			if !strings.HasPrefix(fieldType, "*") {
				fieldType = "*" + fieldType
			}
		}
		tagSuffix := ""
		if !isRequired {
			tagSuffix = ",omitempty"
		}
		tag := fmt.Sprintf("`form:%q json:%q xml:%q`", property.Name+tagSuffix, property.Name+tagSuffix, property.Name+tagSuffix)
		fields = append(fields, fieldName+" "+fieldType+" "+tag)
	}
	return "struct { " + strings.Join(fields, "; ") + " }", true, nil
}
