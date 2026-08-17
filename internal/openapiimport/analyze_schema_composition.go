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
	if len(source.AnyOf) != 2 || hasDirectAllOfSchemaShape(source) {
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
	if value == nil || value.Type == "" && value.Ref == "" {
		return false
	}
	outerDescription := schema.Description
	outerExtensions := schema.Extensions
	*schema = *value
	schema.Nullable = true
	if outerDescription != "" {
		schema.Description = outerDescription
	}
	if len(outerExtensions) > 0 {
		schema.Extensions = outerExtensions
	}
	return true
}

func (a *analyzer) schemaAllOf(schema *Schema, source *base.Schema, path string) bool {
	if len(source.AllOf) == 0 {
		return false
	}
	if len(source.AllOf) == 1 && !hasDirectAllOfSchemaShape(source) {
		proxy := source.AllOf[0]
		if proxy != nil && proxy.IsReference() {
			ref := proxy.GetReference()
			if strings.HasPrefix(ref, "#/components/schemas/") {
				schema.Ref = ref
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

func hasDirectAllOfSchemaShape(source *base.Schema) bool {
	return len(source.Type) > 0 || orderedmap.Len(source.Properties) > 0 || len(source.Required) > 0 ||
		source.Items != nil || source.AdditionalProperties != nil || len(source.Enum) > 0 || source.Format != "" ||
		source.Pattern != "" || source.Minimum != nil || source.Maximum != nil || source.ExclusiveMinimum != nil ||
		source.ExclusiveMaximum != nil || source.MinLength != nil || source.MaxLength != nil ||
		source.MinItems != nil || source.MaxItems != nil || source.Default != nil
}
