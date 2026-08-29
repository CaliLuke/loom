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
		Discriminator        *InlineDiscriminator     `json:"discriminator,omitempty"`
		Items                *InlineSchema            `json:"items,omitempty"`
		AdditionalProperties any                      `json:"additionalProperties,omitempty"`
		Enum                 []any                    `json:"enum,omitempty"`
		Default              any                      `json:"default,omitempty"`
		Minimum              *float64                 `json:"minimum,omitempty"`
		Maximum              *float64                 `json:"maximum,omitempty"`
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
)

// InlineJSONSchema returns a compact JSON Schema for the given Loom attribute.
// The schema is fully resolved and does not include $ref references.
func InlineJSONSchema(attr *AttributeExpr) ([]byte, error) {
	if attr == nil || attr.Type == nil || attr.Type == Empty {
		return json.Marshal(&InlineSchema{
			Type:                 jsonTypeObject,
			AdditionalProperties: false,
		})
	}
	schema, err := buildInlineJSONSchema(attr, make(map[any]struct{}))
	if err != nil {
		return nil, err
	}
	return json.Marshal(schema)
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
		return inlineWrappedJSONSchema(attr, dt.Attribute(), visited, dt, dt.Name())
	default:
		schema.Type = jsonTypeObject
		schema.AdditionalProperties = false
	}

	return schema, nil
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
			schema.Enum = v.Values
		}
		if v.Minimum != nil {
			schema.Minimum = v.Minimum
		}
		if v.Maximum != nil {
			schema.Maximum = v.Maximum
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
	for _, nat := range *dt {
		property, err := buildInlineJSONSchema(nat.Attribute, visited)
		if err != nil {
			return err
		}
		schema.Properties[nat.Name] = property
	}
	schema.AdditionalProperties = false
	if attr.Validation != nil && len(attr.Validation.Required) > 0 {
		schema.Required = attr.Validation.Required
	}
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
	if attr.DefaultValue != nil {
		schema.Default = CanonicalizeExample(attr, attr.DefaultValue)
	}
	if attr.Validation == nil {
		return
	}
	v := attr.Validation
	if len(v.Values) > 0 {
		schema.Enum = v.Values
	}
	if v.Minimum != nil {
		schema.Minimum = v.Minimum
	}
	if v.Maximum != nil {
		schema.Maximum = v.Maximum
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
	if len(v.Required) > 0 {
		schema.Required = v.Required
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
		return jsonTypeObject
	default:
		return jsonTypeString
	}
}
