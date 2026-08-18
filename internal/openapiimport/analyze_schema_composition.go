package openapiimport

import (
	"fmt"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
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
		if !proxy.IsReference() {
			candidate := proxy.Schema()
			if candidate != nil && len(candidate.Type) == 1 && candidate.Type[0] == "null" {
				continue
			}
		}
		if value != nil {
			return false
		}
		value = a.schema(proxy, fmt.Sprintf("%s/anyOf/%d", path, index))
	}
	if value == nil || value.Type == "" && value.Ref == "" && !value.Unconstrained {
		return false
	}
	outerDescription := schema.Description
	outerExtensions := schema.Extensions
	*schema = *value
	// An unconstrained schema already accepts null, so its explicit null branch
	// is semantically redundant and does not need a separate nullable wrapper.
	schema.Nullable = !schema.Unconstrained
	if outerDescription != "" {
		schema.Description = outerDescription
	}
	if len(outerExtensions) > 0 {
		schema.Extensions = outerExtensions
	}
	return true
}

func schemaNullableReferenceOneOf(schema *Schema, source *base.Schema) bool {
	if len(source.OneOf) != 2 || len(source.AnyOf) > 0 || len(source.AllOf) > 0 || hasDirectAllOfSchemaShape(source) {
		return false
	}
	var ref string
	for _, proxy := range source.OneOf {
		if isNullOnlySchema(proxy) {
			continue
		}
		if ref != "" || !isNonNullableLocalReference(proxy) {
			return false
		}
		ref = proxy.GetReference()
	}
	if ref == "" {
		return false
	}
	schema.Ref = ref
	schema.Nullable = true
	return true
}

func (a *analyzer) schemaUntaggedOneOf(schema *Schema, source *base.Schema, path string) bool {
	if len(source.OneOf) < 2 || len(source.AnyOf) > 0 || len(source.AllOf) > 0 || hasDirectAllOfSchemaShape(source) {
		return false
	}
	branches := make([]*Schema, 0, len(source.OneOf))
	for index, proxy := range source.OneOf {
		if proxy == nil || !isObjectSchemaProxy(proxy) {
			return false
		}
		branch := a.schema(proxy, fmt.Sprintf("%s/oneOf/%d", path, index))
		if branch == nil || branch.unsupportedComposition {
			return false
		}
		branches = append(branches, branch)
	}
	schema.OneOf = branches
	return true
}

func isObjectSchemaProxy(proxy *base.SchemaProxy) bool {
	if proxy == nil {
		return false
	}
	if proxy.IsReference() && !isNonNullableLocalReference(proxy) {
		return false
	}
	resolved := proxy.Schema()
	if resolved == nil || resolved.Nullable != nil && *resolved.Nullable {
		return false
	}
	return len(resolved.Type) == 1 && resolved.Type[0] == "object" ||
		len(resolved.Type) == 0 && orderedmap.Len(resolved.Properties) > 0
}

func isNullOnlySchema(proxy *base.SchemaProxy) bool {
	if proxy == nil || proxy.IsReference() {
		return false
	}
	schema := proxy.Schema()
	if schema == nil || len(schema.Type) != 1 || schema.Type[0] != "null" {
		return false
	}
	node := proxy.GetValueNode()
	return node != nil && len(node.Content) == 2 && node.Content[0].Value == "type"
}

func isNonNullableLocalReference(proxy *base.SchemaProxy) bool {
	if proxy == nil || !proxy.IsReference() || proxy.IsTransformedRefWithSiblings() ||
		!strings.HasPrefix(proxy.GetReference(), "#/components/schemas/") {
		return false
	}
	resolved := proxy.Schema()
	return resolved != nil && len(resolved.Type) == 1 && resolved.Type[0] != "null" &&
		(resolved.Nullable == nil || !*resolved.Nullable)
}

func (a *analyzer) schemaAllOf(schema *Schema, source *base.Schema, path string) bool {
	if len(source.AllOf) == 0 {
		return false
	}
	if len(source.AllOf) == 1 && !hasUnsupportedSingleReferenceAllOfSiblings(source) {
		proxy := source.AllOf[0]
		if proxy != nil && proxy.IsReference() {
			ref := proxy.GetReference()
			if strings.HasPrefix(ref, "#/components/schemas/") {
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
	var baseSchema, inlineSchema *Schema
	for index, proxy := range source.AllOf {
		if proxy == nil {
			return false
		}
		partPath := fmt.Sprintf("%s/allOf/%d", path, index)
		if proxy.IsReference() {
			ref := proxy.GetReference()
			if !strings.HasPrefix(ref, "#/components/schemas/") || baseSchema != nil {
				return false
			}
			baseSchema = &Schema{Ref: ref}
			continue
		}
		if inlineSchema != nil {
			return false
		}
		inlineSchema = a.schema(proxy, partPath)
	}
	if baseSchema == nil || inlineSchema == nil || inlineSchema.Type != "object" || len(inlineSchema.Bases) > 0 {
		return false
	}
	schema.Type = "object"
	schema.Bases = []*Schema{baseSchema}
	schema.Properties = inlineSchema.Properties
	schema.Required = inlineSchema.Required
	schema.AdditionalProperties = inlineSchema.AdditionalProperties
	if schema.Description == "" {
		schema.Description = inlineSchema.Description
	}
	a.unsupported(
		"schema-allof-flattened",
		path+"/allOf",
		"object inheritance is rendered with Extend and regenerated OpenAPI flattens the composition",
	)
	return true
}

func hasUnsupportedSingleReferenceAllOfSiblings(source *base.Schema) bool {
	return len(source.Type) > 0 || orderedmap.Len(source.Properties) > 0 || len(source.Required) > 0 ||
		source.Items != nil || source.AdditionalProperties != nil || len(source.Enum) > 0 || source.Format != "" ||
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
		source.Items != nil || source.AdditionalProperties != nil || len(source.Enum) > 0 || source.Format != "" ||
		source.Pattern != "" || source.Minimum != nil || source.Maximum != nil || source.ExclusiveMinimum != nil ||
		source.ExclusiveMaximum != nil || source.MinLength != nil || source.MaxLength != nil ||
		source.MinItems != nil || source.MaxItems != nil || source.Default != nil
}
