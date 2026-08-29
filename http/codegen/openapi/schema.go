package openapi

type (
	// Schema represents the JSON Schema vocabulary used by generated OpenAPI
	// documents.
	Schema struct {
		// Schema identifies the JSON Schema dialect for this schema node.
		Schema string `json:"$schema,omitempty" yaml:"$schema,omitempty"`
		// ID identifies this schema node.
		ID string `json:"id,omitempty" yaml:"id,omitempty"`
		// Title is the display name of the schema.
		Title string `json:"title,omitempty" yaml:"title,omitempty"`
		// Type is the JSON value type accepted by the schema.
		Type Type `json:"type,omitempty" yaml:"type,omitempty"`

		// Items describes array members.
		Items *Schema `json:"items,omitempty" yaml:"items,omitempty"`
		// Properties describes named object members.
		Properties map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
		// Defs contains schema-local reusable definitions.
		Defs map[string]*Schema `json:"$defs,omitempty" yaml:"$defs,omitempty"`
		// Description explains the schema contract.
		Description string `json:"description,omitempty" yaml:"description,omitempty"`
		// DefaultValue is the schema default.
		DefaultValue any `json:"default,omitempty" yaml:"default,omitempty"`
		// Example is an example value accepted by the schema.
		Example any `json:"example,omitempty" yaml:"example,omitempty"`

		// ReadOnly marks a value as response-only.
		ReadOnly bool `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
		// WriteOnly marks a value as request-only.
		WriteOnly bool `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`
		// Deprecated marks the schema contract as deprecated.
		Deprecated bool `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
		// ContentEncoding identifies the encoding applied to string content.
		ContentEncoding string `json:"contentEncoding,omitempty" yaml:"contentEncoding,omitempty"`
		// ContentMediaType identifies the media type of encoded string content.
		ContentMediaType string `json:"contentMediaType,omitempty" yaml:"contentMediaType,omitempty"`
		// ContentSchema describes content after decoding ContentEncoding and ContentMediaType.
		ContentSchema *Schema `json:"contentSchema,omitempty" yaml:"contentSchema,omitempty"`
		// Ref references another schema.
		Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`

		// Enum lists the accepted values.
		Enum []any `json:"enum,omitempty" yaml:"enum,omitempty"`
		// Format refines the semantic format of a value.
		Format string `json:"format,omitempty" yaml:"format,omitempty"`
		// Pattern constrains string values with a regular expression.
		Pattern string `json:"pattern,omitempty" yaml:"pattern,omitempty"`
		// ExclusiveMinimum is the exclusive numeric lower bound.
		ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
		// Minimum is the inclusive numeric lower bound.
		Minimum *float64 `json:"minimum,omitempty" yaml:"minimum,omitempty"`
		// ExclusiveMaximum is the exclusive numeric upper bound.
		ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
		// Maximum is the inclusive numeric upper bound.
		Maximum *float64 `json:"maximum,omitempty" yaml:"maximum,omitempty"`
		// MinLength is the minimum string length.
		MinLength *int `json:"minLength,omitempty" yaml:"minLength,omitempty"`
		// MaxLength is the maximum string length.
		MaxLength *int `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
		// MinItems is the minimum array length.
		MinItems *int `json:"minItems,omitempty" yaml:"minItems,omitempty"`
		// MaxItems is the maximum array length.
		MaxItems *int `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
		// Required lists required object property names.
		Required []string `json:"required,omitempty" yaml:"required,omitempty"`
		// AdditionalProperties controls values for otherwise unnamed properties.
		AdditionalProperties any `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
		// UnevaluatedProperties controls properties not covered by another keyword.
		UnevaluatedProperties any `json:"unevaluatedProperties,omitempty" yaml:"unevaluatedProperties,omitempty"`

		// AllOf requires every listed schema to match.
		AllOf []*Schema `json:"allOf,omitempty" yaml:"allOf,omitempty"`
		// AnyOf requires at least one listed schema to match.
		AnyOf []*Schema `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
		// OneOf requires exactly one listed schema to match.
		OneOf []*Schema `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
		// Discriminator describes union branch selection.
		Discriminator *Discriminator `json:"discriminator,omitempty" yaml:"discriminator,omitempty"`
		// XML describes XML serialization metadata.
		XML *XML `json:"xml,omitempty" yaml:"xml,omitempty"`

		// Extensions contains OpenAPI extension properties.
		Extensions map[string]any `json:"-" yaml:"-"`
	}

	// Type is the JSON type enum.
	Type string

	// Discriminator represents an OpenAPI discriminator object.
	Discriminator struct {
		// PropertyName names the object property that selects the union branch.
		PropertyName string `json:"propertyName" yaml:"propertyName"`
		// Mapping maps discriminator values to schema references.
		Mapping map[string]string `json:"mapping,omitempty" yaml:"mapping,omitempty"`
		// DefaultMapping identifies the fallback schema reference.
		DefaultMapping string `json:"defaultMapping,omitempty" yaml:"defaultMapping,omitempty"`
		// Optional records that the discriminator property may be omitted in OpenAPI 3.2.
		Optional bool `json:"-" yaml:"-"`
	}

	// XML describes how a schema maps to XML nodes.
	XML struct {
		// Name overrides the XML node name.
		Name string `json:"name,omitempty" yaml:"name,omitempty"`
		// Namespace is the absolute XML namespace URI.
		Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
		// Prefix is the XML namespace prefix.
		Prefix string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
		// NodeType identifies the OpenAPI 3.2 XML node kind.
		NodeType string `json:"nodeType,omitempty" yaml:"nodeType,omitempty"`
		// Attribute emits the value as an XML attribute.
		Attribute bool `json:"attribute,omitempty" yaml:"attribute,omitempty"`
		// Wrapped emits an array through a wrapping XML element.
		Wrapped bool `json:"wrapped,omitempty" yaml:"wrapped,omitempty"`
	}

	// _Schema avoids recursively invoking Schema.MarshalJSON.
	_Schema Schema
)

const (
	// Array represents a JSON array.
	Array Type = "array"
	// Boolean represents a JSON boolean.
	Boolean = "boolean"
	// Integer represents a JSON number without a fraction or exponent part.
	Integer = "integer"
	// Number represents any JSON number. Number includes integer.
	Number = "number"
	// Null represents the JSON null value.
	Null = "null"
	// Object represents a JSON object.
	Object = "object"
	// String represents a JSON string.
	String = "string"
)

// NewSchema instantiates an empty schema with writable property and definition
// maps.
func NewSchema() *Schema {
	return &Schema{
		Properties: make(map[string]*Schema),
		Defs:       make(map[string]*Schema),
	}
}

// MarshalJSON returns the JSON encoding of s, including extensions.
func (s *Schema) MarshalJSON() ([]byte, error) {
	return MarshalJSON((*_Schema)(s), s.Extensions)
}

// MarshalYAML returns the YAML representation of s, including extensions.
func (s *Schema) MarshalYAML() (any, error) {
	return MarshalYAML((*_Schema)(s), s.Extensions)
}
