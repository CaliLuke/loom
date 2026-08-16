package openapiimport

import (
	"fmt"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	yaml4 "go.yaml.in/yaml/v4"
)

func (a *analyzer) mediaExamples(media *v3.MediaType, schema *Schema, path string) []Example {
	var examples []Example
	if media.Example != nil {
		if value, ok := a.exampleValue(media.Example, schema, path+"/example"); ok {
			examples = append(examples, Example{Value: value})
		}
	}
	for name, source := range media.Examples.FromOldest() {
		examplePath := path + "/examples/" + escapeJSONPointer(name)
		if source == nil || source.Reference != "" || source.ExternalValue != "" || source.DataValue != nil || source.SerializedValue != "" {
			a.unsupported("examples", examplePath, "media example cannot be expressed as a Loom Example value")
			continue
		}
		if orderedmap.Len(source.Extensions) > 0 {
			a.unsupported("examples", examplePath, "media example extensions are not in the strict import subset")
		}
		if value, ok := a.exampleValue(source.Value, schema, examplePath+"/value"); ok {
			examples = append(examples, Example{
				Name: name, Summary: source.Summary, Description: source.Description, Value: value,
			})
		}
	}
	return examples
}

func (a *analyzer) schemaExamples(schema *Schema, source *base.Schema, path string) {
	if source.Example != nil {
		if value, ok := a.exampleValue(source.Example, schema, path+"/example"); ok {
			schema.Examples = append(schema.Examples, Example{Value: value})
		}
	}
	for index, node := range source.Examples {
		if value, ok := a.exampleValue(node, schema, fmt.Sprintf("%s/examples/%d", path, index)); ok {
			schema.Examples = append(schema.Examples, Example{
				Name:  fmt.Sprintf("example-%d", index+1),
				Value: value,
			})
		}
	}
}

func (a *analyzer) exampleValue(node *yaml4.Node, schema *Schema, path string) (any, bool) {
	if node == nil {
		return nil, false
	}
	var value any
	if err := node.Decode(&value); err != nil {
		a.unsupported("examples", path, fmt.Sprintf("example value cannot be decoded: %v", err))
		return nil, false
	}
	if value == nil {
		if schema != nil && schema.Nullable {
			return nil, true
		}
		a.unsupported("examples", path, "null example requires a nullable schema")
		return nil, false
	}
	if _, err := exampleLiteral(value); err != nil {
		a.unsupported("examples", path, err.Error())
		return nil, false
	}
	if !exampleCompatibleWithSchema(schema, value) {
		a.unsupported("examples", path, "example value is not compatible with its schema")
		return nil, false
	}
	return value, true
}

func exampleCompatibleWithSchema(schema *Schema, value any) bool {
	if value == nil {
		return schema != nil && schema.Nullable
	}
	if schema == nil || schema.Ref != "" {
		return true
	}
	switch schema.Type {
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "integer":
		return isExampleInteger(value)
	case "number":
		return isExampleInteger(value) || isExampleFloat(value)
	case "array":
		values, ok := value.([]any)
		if !ok {
			return false
		}
		for _, item := range values {
			if !exampleCompatibleWithSchema(schema.Items, item) {
				return false
			}
		}
		return true
	case "object":
		return exampleObjectCompatible(schema, value)
	default:
		return false
	}
}

func exampleObjectCompatible(schema *Schema, value any) bool {
	values, ok := value.(map[string]any)
	if !ok {
		return false
	}
	properties := make(map[string]*Schema, len(schema.Properties))
	for _, property := range schema.Properties {
		properties[property.Name] = property.Schema
	}
	for _, name := range schema.Required {
		if _, ok := values[name]; !ok {
			return false
		}
	}
	for name, item := range values {
		property, defined := properties[name]
		if defined && !exampleCompatibleWithSchema(property, item) {
			return false
		}
		if !defined && schema.AdditionalProperties != nil && schema.AdditionalProperties.Allowed != nil && !*schema.AdditionalProperties.Allowed {
			return false
		}
	}
	return true
}

func isExampleInteger(value any) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

func isExampleFloat(value any) bool {
	switch value.(type) {
	case float32, float64:
		return true
	default:
		return false
	}
}
