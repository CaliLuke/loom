package expr

import (
	"encoding/json/v2"
	"fmt"
)

type (
	// InlineSchema represents a fully inlined JSON Schema document derived from
	// a Loom attribute. It avoids $ref indirections so machine consumers can
	// consume standalone payload and result contracts directly.
	//
	//nolint:tagliatelle // JSON Schema uses camelCase field names.
	InlineSchema struct {
		Type                 string                   `json:"type,omitempty"`
		Title                string                   `json:"title,omitempty"`
		Description          string                   `json:"description,omitempty"`
		Examples             []any                    `json:"examples,omitempty"`
		Required             []string                 `json:"required,omitempty"`
		Properties           map[string]*InlineSchema `json:"properties,omitempty"`
		OneOf                []*InlineSchema          `json:"oneOf,omitempty"`
		AnyOf                []*InlineSchema          `json:"anyOf,omitempty"`
		AllOf                []*InlineSchema          `json:"allOf,omitempty"`
		Discriminator        *InlineDiscriminator     `json:"discriminator,omitempty"`
		Items                *InlineSchema            `json:"items,omitempty"`
		AdditionalProperties any                      `json:"additionalProperties,omitempty"`
		Enum                 []any                    `json:"enum,omitempty"`
		Default              any                      `json:"default,omitempty"`
		Minimum              any                      `json:"minimum,omitempty"`
		Maximum              any                      `json:"maximum,omitempty"`
		ExclusiveMinimum     any                      `json:"exclusiveMinimum,omitempty"`
		ExclusiveMaximum     any                      `json:"exclusiveMaximum,omitempty"`
		MinLength            *int                     `json:"minLength,omitempty"`
		MaxLength            *int                     `json:"maxLength,omitempty"`
		Pattern              string                   `json:"pattern,omitempty"`
		Format               string                   `json:"format,omitempty"`
		MinItems             *int                     `json:"minItems,omitempty"`
		MaxItems             *int                     `json:"maxItems,omitempty"`
	}

	// InlineDiscriminator mirrors the JSON Schema/OpenAPI discriminator object.
	//
	//nolint:tagliatelle // JSON Schema uses camelCase field names.
	InlineDiscriminator struct {
		PropertyName string `json:"propertyName,omitempty"`
	}
)

const (
	jsonTypeObject  = "object"
	jsonTypeArray   = "array"
	jsonTypeString  = "string"
	jsonTypeInteger = "integer"
	jsonTypeNumber  = "number"
	jsonTypeBoolean = "boolean"
	jsonTypeNull    = "null"
)

// InlineJSONSchema returns a compact JSON Schema for the given Loom attribute.
// The schema is fully resolved and does not include $ref references.
func InlineJSONSchema(attr *AttributeExpr) ([]byte, error) {
	if attr == nil || attr.Type == nil || attr.Type == Empty {
		return json.Marshal(&InlineSchema{
			Type:                 jsonTypeObject,
			AdditionalProperties: false,
		}, json.Deterministic(true))
	}
	schema, err := buildInlineJSONSchema(attr, make(map[any]struct{}))
	if err != nil {
		return nil, err
	}
	return json.Marshal(schema, json.Deterministic(true))
}

func buildInlineJSONSchema(attr *AttributeExpr, visited map[any]struct{}) (*InlineSchema, error) {
	if attr == nil || attr.Type == nil {
		return &InlineSchema{
			Type:                 jsonTypeObject,
			AdditionalProperties: false,
		}, nil
	}

	schema := &InlineSchema{
		Title:       attr.Title,
		Description: attr.Description,
	}
	populateInlineSchemaMetadata(schema, attr)

	switch dt := attr.Type.(type) {
	case Primitive:
		schema.Type = primitiveToInlineJSONType(dt)
		applyInlinePrimitiveBounds(schema, dt)
	case *Array:
		if err := populateInlineArraySchema(schema, attr, dt, visited); err != nil {
			return nil, err
		}
	case *Map:
		if err := populateInlineMapSchema(schema, dt, visited); err != nil {
			return nil, err
		}
	case *Union:
		if err := populateInlineUnionSchema(schema, dt, visited); err != nil {
			return nil, err
		}
	case *Object:
		if err := populateInlineObjectSchema(schema, attr, dt, visited); err != nil {
			return nil, err
		}
	case UserType:
		schema, err := inlineWrappedJSONSchema(attr, dt.Attribute(), visited, dt, dt.Name())
		if err != nil {
			return nil, err
		}
		if attr.Nullable && !IsNullable(dt.Attribute()) {
			return nullableInlineJSONSchema(schema), nil
		}
		return schema, nil
	default:
		schema.Type = jsonTypeObject
		schema.AdditionalProperties = false
	}

	if attr.Nullable {
		return nullableInlineJSONSchema(schema), nil
	}
	return schema, nil
}

func nullableInlineJSONSchema(schema *InlineSchema) *InlineSchema {
	return &InlineSchema{AnyOf: []*InlineSchema{schema, {Type: jsonTypeNull}}}
}

func populateInlineSchemaMetadata(schema *InlineSchema, attr *AttributeExpr) {
	if examples := attr.ExtractUserExamples(); len(examples) > 0 {
		schema.Examples = make([]any, 0, len(examples))
		for _, example := range examples {
			if example == nil {
				continue
			}
			schema.Examples = append(schema.Examples, CanonicalizeExample(attr, example.Value))
		}
	}
	if attr.DefaultValue != nil {
		schema.Default = CanonicalizeExample(attr, attr.DefaultValue)
	}
	if v := attr.Validation; v != nil {
		if len(v.Values) > 0 {
			schema.Enum = canonicalizeInlineValues(attr, v.Values)
		}
		if v.Minimum != nil {
			schema.Minimum = v.Minimum
		}
		if v.Maximum != nil {
			schema.Maximum = v.Maximum
		}
		if v.ExclusiveMinimum != nil {
			schema.ExclusiveMinimum = v.ExclusiveMinimum
		}
		if v.ExclusiveMaximum != nil {
			schema.ExclusiveMaximum = v.ExclusiveMaximum
		}
		if v.MinLength != nil {
			schema.MinLength = v.MinLength
		}
		if v.MaxLength != nil {
			schema.MaxLength = v.MaxLength
		}
		if v.Pattern != "" {
			schema.Pattern = v.Pattern
		}
		if v.Format != "" {
			schema.Format = string(v.Format)
		}
	}
}

func populateInlineArraySchema(schema *InlineSchema, attr *AttributeExpr, dt *Array, visited map[any]struct{}) error {
	schema.Type = jsonTypeArray
	if dt.ElemType != nil {
		items, err := buildInlineJSONSchema(dt.ElemType, visited)
		if err != nil {
			return err
		}
		schema.Items = items
	}
	if v := attr.Validation; v != nil {
		if v.MinLength != nil {
			schema.MinItems = v.MinLength
			schema.MinLength = nil
		}
		if v.MaxLength != nil {
			schema.MaxItems = v.MaxLength
			schema.MaxLength = nil
		}
	}
	return nil
}

func populateInlineMapSchema(schema *InlineSchema, dt *Map, visited map[any]struct{}) error {
	schema.Type = jsonTypeObject
	if dt.ElemType != nil {
		properties, err := buildInlineJSONSchema(dt.ElemType, visited)
		if err != nil {
			return err
		}
		schema.AdditionalProperties = properties
	} else {
		schema.AdditionalProperties = true
	}
	return nil
}

func populateInlineUnionSchema(schema *InlineSchema, dt *Union, visited map[any]struct{}) error {
	if dt.Untagged {
		schema.OneOf = make([]*InlineSchema, 0, len(dt.Values))
		for _, val := range dt.Values {
			valueSchema, err := buildInlineJSONSchema(val.Attribute, visited)
			if err != nil {
				return err
			}
			schema.OneOf = append(schema.OneOf, valueSchema)
		}
		return nil
	}
	typeKey := dt.GetTypeKey()
	valueKey := dt.GetValueKey()
	schema.Type = jsonTypeObject
	schema.OneOf = make([]*InlineSchema, 0, len(dt.Values))
	schema.Discriminator = &InlineDiscriminator{PropertyName: typeKey}
	for _, val := range dt.Values {
		valueSchema, err := buildInlineJSONSchema(val.Attribute, visited)
		if err != nil {
			return err
		}
		schema.OneOf = append(schema.OneOf, &InlineSchema{
			Type: jsonTypeObject,
			Properties: map[string]*InlineSchema{
				typeKey: {
					Type: jsonTypeString,
					Enum: []any{UnionVariantTag(val)},
				},
				valueKey: valueSchema,
			},
			Required:             []string{typeKey, valueKey},
			AdditionalProperties: false,
		})
	}
	return nil
}

func populateInlineObjectSchema(schema *InlineSchema, attr *AttributeExpr, dt *Object, visited map[any]struct{}) error {
	schema.Type = jsonTypeObject
	schema.Properties = make(map[string]*InlineSchema, len(*dt))
	designNames := make(map[string]struct{}, len(*dt))
	for _, nat := range *dt {
		designNames[nat.Name] = struct{}{}
	}
	for _, nat := range *dt {
		name := JSONFieldName(nat.Name, nat.Attribute)
		if name == "-" {
			continue
		}
		if name == "" {
			return fmt.Errorf("object field %q cannot use an empty JSON tag name", nat.Name)
		}
		if _, conflicts := designNames[name]; conflicts && name != nat.Name {
			return fmt.Errorf("object field %q JSON name %q conflicts with another design field name", nat.Name, name)
		}
		if _, exists := schema.Properties[name]; exists {
			return fmt.Errorf("object has duplicate JSON field name %q", name)
		}
		property, err := buildInlineJSONSchema(nat.Attribute, visited)
		if err != nil {
			return err
		}
		schema.Properties[name] = property
		if attr.IsRequired(nat.Name) {
			schema.Required = append(schema.Required, name)
		}
	}
	schema.AdditionalProperties = false
	return nil
}

func inlineWrappedJSONSchema(wrapper *AttributeExpr, inner *AttributeExpr, visited map[any]struct{}, identity any, typeName string) (*InlineSchema, error) {
	if _, ok := visited[identity]; ok {
		return nil, fmt.Errorf("recursive user type %q cannot be converted to inline JSON Schema", typeName)
	}
	visited[identity] = struct{}{}
	defer delete(visited, identity)

	schema, err := buildInlineJSONSchema(inner, visited)
	if err != nil {
		return nil, err
	}
	if wrapper.Validation != nil {
		schema = &InlineSchema{Type: schema.Type, AllOf: []*InlineSchema{schema}}
	}
	applyInlineWrapperMetadata(schema, wrapper)
	return schema, nil
}

func applyInlineWrapperMetadata(schema *InlineSchema, attr *AttributeExpr) {
	if schema == nil || attr == nil {
		return
	}
	if attr.Description != "" {
		schema.Description = attr.Description
	}
	if attr.Title != "" {
		schema.Title = attr.Title
	}
	if examples := attr.ExtractUserExamples(); len(examples) > 0 {
		schema.Examples = make([]any, 0, len(examples))
		for _, example := range examples {
			if example != nil {
				schema.Examples = append(schema.Examples, CanonicalizeExample(attr, example.Value))
			}
		}
	}
	if attr.DefaultValue != nil {
		schema.Default = CanonicalizeExample(attr, attr.DefaultValue)
	}
	applyInlineValidation(schema, attr)
}

func applyInlineValidation(schema *InlineSchema, attr *AttributeExpr) {
	if attr.Validation == nil {
		return
	}
	v := attr.Validation
	if len(v.Values) > 0 {
		schema.Enum = canonicalizeInlineValues(attr, v.Values)
	}
	if v.Minimum != nil {
		schema.Minimum = v.Minimum
	}
	if v.Maximum != nil {
		schema.Maximum = v.Maximum
	}
	if v.ExclusiveMinimum != nil {
		schema.ExclusiveMinimum = v.ExclusiveMinimum
	}
	if v.ExclusiveMaximum != nil {
		schema.ExclusiveMaximum = v.ExclusiveMaximum
	}
	if schema.Type == jsonTypeArray {
		if v.MinLength != nil {
			schema.MinItems = v.MinLength
			schema.MinLength = nil
		}
		if v.MaxLength != nil {
			schema.MaxItems = v.MaxLength
			schema.MaxLength = nil
		}
	} else {
		if v.MinLength != nil {
			schema.MinLength = v.MinLength
		}
		if v.MaxLength != nil {
			schema.MaxLength = v.MaxLength
		}
	}
	if v.Pattern != "" {
		schema.Pattern = v.Pattern
	}
	if v.Format != "" {
		schema.Format = string(v.Format)
	}
	if len(v.Required) > 0 {
		schema.Required = inlineRequiredNames(attr, v.Required)
	}
}

func applyInlinePrimitiveBounds(schema *InlineSchema, primitive Primitive) {
	var minimum, maximum any
	switch primitive {
	case Int, Int64:
		minimum = int64(-9223372036854775808)
		maximum = int64(9223372036854775807)
	case Int32:
		minimum = int32(-2147483648)
		maximum = int32(2147483647)
	case UInt, UInt64:
		minimum = uint64(0)
		maximum = uint64(18446744073709551615)
	case UInt32:
		minimum = uint32(0)
		maximum = uint32(4294967295)
	}
	if minimum != nil && inlineBoundOutsideMinimum(schema.Minimum, minimum) {
		schema.Minimum = minimum
	}
	if maximum != nil && inlineBoundOutsideMaximum(schema.Maximum, maximum) {
		schema.Maximum = maximum
	}
}

func inlineBoundOutsideMinimum(current, domain any) bool {
	if current == nil {
		return true
	}
	currentNumber, currentOK := inlineBoundFloat64(current)
	domainNumber, domainOK := inlineBoundFloat64(domain)
	return !currentOK || !domainOK || currentNumber <= domainNumber
}

func inlineBoundOutsideMaximum(current, domain any) bool {
	if current == nil {
		return true
	}
	currentNumber, currentOK := inlineBoundFloat64(current)
	domainNumber, domainOK := inlineBoundFloat64(domain)
	return !currentOK || !domainOK || currentNumber >= domainNumber
}

func inlineBoundFloat64(value any) (float64, bool) {
	switch actual := value.(type) {
	case *float64:
		return *actual, true
	case float64:
		return actual, true
	case float32:
		return float64(actual), true
	case int:
		return float64(actual), true
	case int32:
		return float64(actual), true
	case int64:
		return float64(actual), true
	case uint:
		return float64(actual), true
	case uint32:
		return float64(actual), true
	case uint64:
		return float64(actual), true
	default:
		return 0, false
	}
}

func primitiveToInlineJSONType(p Primitive) string {
	switch p {
	case Boolean:
		return jsonTypeBoolean
	case Int, Int32, Int64, UInt, UInt32, UInt64:
		return jsonTypeInteger
	case Float32, Float64:
		return jsonTypeNumber
	case String, Bytes:
		return jsonTypeString
	case Any:
		return ""
	default:
		return jsonTypeString
	}
}

func inlineRequiredNames(attribute *AttributeExpr, required []string) []string {
	resolved := unwrapUserTypeAttr(attribute)
	if resolved == nil {
		return required
	}
	object, ok := resolved.Type.(*Object)
	if !ok {
		return required
	}
	names := make([]string, 0, len(required))
	for _, requiredName := range required {
		wireName := requiredName
		for _, field := range *object {
			if field != nil && field.Name == requiredName {
				wireName = JSONFieldName(field.Name, field.Attribute)
				break
			}
		}
		if wireName != "-" {
			names = append(names, wireName)
		}
	}
	return names
}

func canonicalizeInlineValues(attribute *AttributeExpr, values []any) []any {
	canonical := make([]any, len(values))
	for index, value := range values {
		canonical[index] = CanonicalizeExample(attribute, value)
	}
	return canonical
}
