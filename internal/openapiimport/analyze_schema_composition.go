package openapiimport

import (
	"fmt"
	"slices"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"

	"github.com/CaliLuke/loom/internal/openapiref"
)

func nullableUnionType(types []string) (string, bool) {
	if len(types) != 2 {
		return "", false
	}
	if types[0] == "null" && types[1] != "null" {
		return types[1], true
	}
	if types[1] == "null" && types[0] != "null" {
		return types[0], true
	}
	return "", false
}

func (a *analyzer) schemaNullableAnyOf(schema *Schema, source *base.Schema, path string) bool {
	if len(source.AnyOf) != 2 || len(source.OneOf) > 0 || hasDirectAllOfSchemaShape(source) {
		return false
	}
	var value *Schema
	for index, proxy := range source.AnyOf {
		if proxy == nil {
			return false
		}
		if isNullOnlySchema(proxy) {
			continue
		}
		if value != nil {
			return false
		}
		value = a.schema(proxy, fmt.Sprintf("%s/anyOf/%d", path, index))
	}
	if value == nil || value.Type == "" && value.Ref == "" && !value.Unconstrained && len(value.OneOf) == 0 {
		return false
	}
	*schema = *value
	// An unconstrained schema already accepts null, so its explicit null branch
	// is semantically redundant and does not need a separate nullable wrapper.
	schema.Nullable = !schema.Unconstrained
	return true
}

func isNullOnlySchema(proxy *base.SchemaProxy) bool {
	if proxy == nil || proxy.IsReference() {
		return false
	}
	schema := proxy.Schema()
	if schema == nil || len(schema.Type) != 1 || schema.Type[0] != "null" {
		return false
	}
	node := semanticSchemaNode(proxy)
	return node != nil && len(node.Content) == 2 && node.Content[0].Value == "type"
}

func (a *analyzer) schemaAllOf(schema *Schema, source *base.Schema, path string) bool {
	if len(source.AllOf) == 0 {
		return false
	}
	if len(source.AllOf) == 1 && !hasUnsupportedSingleReferenceAllOfSiblings(source) {
		proxy := source.AllOf[0]
		if proxy != nil && proxy.IsReference() && !proxy.IsTransformedRefWithSiblings() {
			ref := proxy.GetReference()
			if _, err := openapiref.Name(ref, "#/components/schemas/"); err == nil {
				schema.Ref = ref
				if hasAllOfValidationSiblings(source) {
					schema.Type = referencedSchemaType(proxy)
				}
				return true
			}
		}
	}
	if len(source.AllOf) != 2 || hasDirectAllOfSchemaShape(source) {
		return false
	}
	var reference *base.SchemaProxy
	var inline *base.Schema
	inlinePath := path
	for index, proxy := range source.AllOf {
		if proxy == nil {
			return false
		}
		if proxy.IsReference() {
			if proxy.IsTransformedRefWithSiblings() {
				return false
			}
			if _, err := openapiref.Name(proxy.GetReference(), "#/components/schemas/"); err != nil || reference != nil {
				return false
			}
			reference = proxy
			continue
		}
		if inline != nil {
			return false
		}
		inline = proxy.Schema()
		inlinePath = fmt.Sprintf("%s/allOf/%d", path, index)
		a.schemaUnknownKeywordDiagnostics(proxy, inlinePath)
	}
	if reference == nil || inline == nil || !slices.Equal(inline.Type, []string{"object"}) {
		return false
	}
	target := referenceSchemaShape(reference, make(map[*base.Schema]bool))
	if target == nil {
		return false
	}
	*schema = *a.schemaObjectReference(inline, target, reference.GetReference(), inlinePath)
	return true
}

func hasUnsupportedSingleReferenceAllOfSiblings(source *base.Schema) bool {
	return len(source.Type) > 0 || orderedmap.Len(source.Properties) > 0 || len(source.Required) > 0 ||
		source.Items != nil || source.AdditionalProperties != nil || hasSchemaEnum(source) || source.Const != nil || source.Format != "" ||
		source.Pattern != "" || source.MinLength != nil || source.MaxLength != nil || source.MinItems != nil ||
		source.MaxItems != nil
}

func hasAllOfValidationSiblings(source *base.Schema) bool {
	return source.Minimum != nil || source.Maximum != nil || source.ExclusiveMinimum != nil ||
		source.ExclusiveMaximum != nil || source.Default != nil
}

func referencedSchemaType(proxy *base.SchemaProxy) string {
	resolved := proxy.Schema()
	if resolved == nil {
		return ""
	}
	if len(resolved.Type) == 1 && resolved.Type[0] != "null" {
		return resolved.Type[0]
	}
	typeName, _ := nullableUnionType(resolved.Type)
	return typeName
}

func hasDirectAllOfSchemaShape(source *base.Schema) bool {
	return len(source.Type) > 0 || orderedmap.Len(source.Properties) > 0 || len(source.Required) > 0 ||
		source.Items != nil || source.AdditionalProperties != nil || hasSchemaEnum(source) || source.Const != nil || source.Format != "" ||
		source.Pattern != "" || source.Minimum != nil || source.Maximum != nil || source.ExclusiveMinimum != nil ||
		source.ExclusiveMaximum != nil || source.MinLength != nil || source.MaxLength != nil ||
		source.MinItems != nil || source.MaxItems != nil || source.Default != nil
}
