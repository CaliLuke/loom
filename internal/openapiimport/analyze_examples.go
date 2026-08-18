package openapiimport

import (
	"fmt"
	"reflect"
	"unicode/utf8"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	yaml4 "go.yaml.in/yaml/v4"

	loom "github.com/CaliLuke/loom/pkg"
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
		if source == nil {
			a.unsupported("examples", examplePath, "media example is nil")
			continue
		}
		if reference := exampleReference(source); reference != "" {
			example, ok := a.referencedExample(reference, schema, examplePath)
			if ok {
				example.Name = name
				examples = append(examples, example)
			}
			continue
		}
		example, ok := a.normalizedExample(source, schema, examplePath)
		if ok {
			example.Name = name
			examples = append(examples, example)
		}
	}
	return examples
}

func (a *analyzer) componentExample(name string, source *base.Example, path string) (Example, bool) {
	if source == nil {
		a.unsupported("examples", path, "component example is nil")
		return Example{}, false
	}
	if exampleReference(source) != "" {
		a.unsupported("examples", path, "component example references are not in the strict import subset")
		return Example{}, false
	}
	example, ok := a.normalizedExample(source, nil, path)
	if !ok {
		return Example{}, false
	}
	example.ComponentName = name
	return example, true
}

func exampleReference(source *base.Example) string {
	if source == nil {
		return ""
	}
	if source.Reference != "" {
		return source.Reference
	}
	if low := source.GoLow(); low != nil && low.IsReference() {
		return low.GetReference()
	}
	return ""
}

func (a *analyzer) referencedExample(reference string, schema *Schema, path string) (Example, bool) {
	name, err := localComponentReferenceName(reference, "#/components/examples/")
	if err != nil {
		a.unsupported("examples", path, "example reference must target a local example component")
		return Example{}, false
	}
	example, ok := a.examples[name]
	if !ok {
		a.unsupported("examples", path, fmt.Sprintf("example component %q does not resolve", name))
		return Example{}, false
	}
	if !a.exampleCompatibleWithSchema(schema, example.Value, make(map[string]struct{})) {
		a.unsupported("examples", path, "referenced example value is not compatible with its schema")
		return Example{}, false
	}
	return example, true
}

func (a *analyzer) normalizedExample(source *base.Example, schema *Schema, path string) (Example, bool) {
	if !a.openAPI32() && (source.DataValue != nil || source.SerializedValue != "") {
		a.unsupported("versioned-field", path, "dataValue and serializedValue require OpenAPI 3.2")
		return Example{}, false
	}
	if source.ExternalValue != "" {
		a.unsupported("examples", path+"/externalValue", "external example values are not in the strict import subset")
		return Example{}, false
	}
	if orderedmap.Len(source.Extensions) > 0 {
		a.unsupported("examples", path, "example extensions are not in the strict import subset")
		return Example{}, false
	}
	if source.Value != nil && source.DataValue != nil {
		a.unsupported("examples", path, "example value and dataValue are mutually exclusive")
		return Example{}, false
	}
	if source.Value != nil && source.SerializedValue != "" {
		a.unsupported("examples", path, "example value and serializedValue are mutually exclusive")
		return Example{}, false
	}
	node := source.Value
	dataValue := source.DataValue != nil
	valuePath := path + "/value"
	if dataValue {
		node = source.DataValue
		valuePath = path + "/dataValue"
	}
	if node == nil {
		a.unsupported("examples", path, "example needs value or dataValue to preserve its typed meaning")
		return Example{}, false
	}
	value, ok := a.exampleValue(node, schema, valuePath)
	if !ok {
		return Example{}, false
	}
	return Example{
		Summary: source.Summary, Description: source.Description, Value: value,
		DataValue: dataValue, SerializedValue: source.SerializedValue,
	}, true
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
		if a.exampleCompatibleWithSchema(schema, nil, make(map[string]struct{})) {
			return nil, true
		}
		a.unsupported("examples", path, "null example requires a nullable schema")
		return nil, false
	}
	if _, err := exampleLiteral(value); err != nil {
		a.unsupported("examples", path, err.Error())
		return nil, false
	}
	if !a.exampleCompatibleWithSchema(schema, value, make(map[string]struct{})) {
		a.unsupported("examples", path, "example value is not compatible with its schema")
		return nil, false
	}
	return value, true
}

func (a *analyzer) exampleCompatibleWithSchema(schema *Schema, value any, seen map[string]struct{}) bool {
	if schema == nil {
		return value != nil
	}
	if value == nil && schema.Nullable {
		return true
	}
	if schema.Ref != "" {
		return a.referencedSchemaExampleCompatible(schema.Ref, value, seen)
	}
	if value == nil {
		return false
	}
	if len(schema.OneOf) > 0 {
		return a.oneOfExampleCompatible(schema.OneOf, value, seen)
	}
	if !normalizedExampleValidationMatches(schema, value) {
		return false
	}
	return a.exampleTypeCompatible(schema, value)
}

func (a *analyzer) referencedSchemaExampleCompatible(ref string, value any, seen map[string]struct{}) bool {
	name, err := localComponentReferenceName(ref, "#/components/schemas/")
	if err != nil {
		return false
	}
	if _, exists := seen[name]; exists {
		return false
	}
	resolved, ok := a.schemas[name]
	if !ok || resolved == nil {
		return true
	}
	seen[name] = struct{}{}
	matched := a.exampleCompatibleWithSchema(resolved, value, seen)
	delete(seen, name)
	return matched
}

func (a *analyzer) oneOfExampleCompatible(branches []*Schema, value any, seen map[string]struct{}) bool {
	matches := 0
	for _, branch := range branches {
		if a.exampleCompatibleWithSchema(branch, value, seen) {
			matches++
		}
	}
	return matches == 1
}

func (a *analyzer) exampleTypeCompatible(schema *Schema, value any) bool {
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
			if !a.exampleCompatibleWithSchema(schema.Items, item, make(map[string]struct{})) {
				return false
			}
		}
		return true
	case "object":
		return a.exampleObjectCompatible(schema, value)
	default:
		return false
	}
}

func (a *analyzer) exampleObjectCompatible(schema *Schema, value any) bool {
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
		if defined && !a.exampleCompatibleWithSchema(property, item, make(map[string]struct{})) {
			return false
		}
		if !defined && schema.AdditionalProperties != nil && schema.AdditionalProperties.Allowed != nil && !*schema.AdditionalProperties.Allowed {
			return false
		}
	}
	return true
}

func normalizedExampleValidationMatches(schema *Schema, value any) bool {
	if len(schema.Enum) > 0 {
		matched := false
		for _, candidate := range schema.Enum {
			if normalizedExampleValuesEqual(candidate, value) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if stringValue, ok := value.(string); ok {
		if schema.Pattern != "" && loom.ValidatePattern("example", stringValue, schema.Pattern) != nil {
			return false
		}
		if _, recognized := stringFormatDSL(schema.Format); recognized &&
			loom.ValidateFormat("example", stringValue, loom.Format(schema.Format)) != nil {
			return false
		}
		length := int64(utf8.RuneCountInString(stringValue))
		if schema.MinLength != nil && length < *schema.MinLength {
			return false
		}
		if schema.MaxLength != nil && length > *schema.MaxLength {
			return false
		}
	}
	number, numeric := normalizedExampleNumber(value)
	if !numeric {
		return true
	}
	if schema.Minimum != nil && number < *schema.Minimum {
		return false
	}
	if schema.ExclusiveMinimum != nil && number <= *schema.ExclusiveMinimum {
		return false
	}
	if schema.Maximum != nil && number > *schema.Maximum {
		return false
	}
	return schema.ExclusiveMaximum == nil || number < *schema.ExclusiveMaximum
}

func normalizedExampleValuesEqual(left, right any) bool {
	leftNumber, leftNumeric := normalizedExampleNumber(left)
	rightNumber, rightNumeric := normalizedExampleNumber(right)
	if leftNumeric && rightNumeric {
		return leftNumber == rightNumber
	}
	return reflect.DeepEqual(left, right)
}

func normalizedExampleNumber(value any) (float64, bool) {
	switch actual := value.(type) {
	case int:
		return float64(actual), true
	case int8:
		return float64(actual), true
	case int16:
		return float64(actual), true
	case int32:
		return float64(actual), true
	case int64:
		return float64(actual), true
	case uint:
		return float64(actual), true
	case uint8:
		return float64(actual), true
	case uint16:
		return float64(actual), true
	case uint32:
		return float64(actual), true
	case uint64:
		return float64(actual), true
	case float32:
		return float64(actual), true
	case float64:
		return actual, true
	default:
		return 0, false
	}
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
