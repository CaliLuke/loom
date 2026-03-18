package expr

import "encoding/json"

type (
	// InlineSchema represents a fully inlined JSON Schema document derived from
	// a Goa attribute. It avoids $ref indirections so machine consumers can
	// consume standalone payload and result contracts directly.
	//
	//nolint:tagliatelle // JSON Schema uses camelCase field names.
	InlineSchema struct {
		Type                 string                   `json:"type,omitempty"`
		Description          string                   `json:"description,omitempty"`
		Examples             []any                    `json:"examples,omitempty"`
		Required             []string                 `json:"required,omitempty"`
		Properties           map[string]*InlineSchema `json:"properties,omitempty"`
		OneOf                []*InlineSchema          `json:"oneOf,omitempty"`
		AnyOf                []*InlineSchema          `json:"anyOf,omitempty"`
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
)

const (
	jsonTypeObject  = "object"
	jsonTypeArray   = "array"
	jsonTypeString  = "string"
	jsonTypeInteger = "integer"
	jsonTypeNumber  = "number"
	jsonTypeBoolean = "boolean"
)

// InlineJSONSchema returns a compact JSON Schema for the given Goa attribute.
// The schema is fully resolved and does not include $ref references.
func InlineJSONSchema(attr *AttributeExpr) ([]byte, error) {
	if attr == nil || attr.Type == nil || attr.Type == Empty {
		return json.Marshal(&InlineSchema{
			Type:                 jsonTypeObject,
			AdditionalProperties: false,
		})
	}
	return json.Marshal(buildInlineJSONSchema(attr))
}

func buildInlineJSONSchema(attr *AttributeExpr) *InlineSchema {
	if attr == nil || attr.Type == nil {
		return &InlineSchema{
			Type:                 jsonTypeObject,
			AdditionalProperties: false,
		}
	}

	schema := &InlineSchema{
		Description: attr.Description,
	}
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

	switch dt := attr.Type.(type) {
	case Primitive:
		schema.Type = primitiveToInlineJSONType(dt)
	case *Array:
		schema.Type = jsonTypeArray
		if dt.ElemType != nil {
			schema.Items = buildInlineJSONSchema(dt.ElemType)
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
	case *Map:
		schema.Type = jsonTypeObject
		if dt.ElemType != nil {
			schema.AdditionalProperties = buildInlineJSONSchema(dt.ElemType)
		} else {
			schema.AdditionalProperties = true
		}
	case *Union:
		typeKey := dt.GetTypeKey()
		valueKey := dt.GetValueKey()
		schema.Type = jsonTypeObject
		schema.OneOf = make([]*InlineSchema, 0, len(dt.Values))
		for _, val := range dt.Values {
			schema.OneOf = append(schema.OneOf, &InlineSchema{
				Type: jsonTypeObject,
				Properties: map[string]*InlineSchema{
					typeKey: {
						Type: jsonTypeString,
						Enum: []any{UnionVariantTag(val)},
					},
					valueKey: buildInlineJSONSchema(val.Attribute),
				},
				Required:             []string{typeKey, valueKey},
				AdditionalProperties: false,
			})
		}
	case *Object:
		schema.Type = jsonTypeObject
		schema.Properties = make(map[string]*InlineSchema, len(*dt))
		for _, nat := range *dt {
			schema.Properties[nat.Name] = buildInlineJSONSchema(nat.Attribute)
		}
		schema.AdditionalProperties = false
		if attr.Validation != nil && len(attr.Validation.Required) > 0 {
			schema.Required = attr.Validation.Required
		}
	case UserType:
		return buildInlineJSONSchema(dt.Attribute())
	default:
		schema.Type = jsonTypeObject
		schema.AdditionalProperties = false
	}

	return schema
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
