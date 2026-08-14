// Package openapiimport analyzes OpenAPI documents into a deterministic,
// transport-neutral model suitable for later Loom design generation.
package openapiimport

// Document is the normalized subset of an OpenAPI document understood by the
// importer.
type Document struct {
	// OpenAPIVersion is the exact supported 3.x version declared by the input.
	OpenAPIVersion string
	// Title is the API title.
	Title string
	// Description is the API description.
	Description string
	// APIVersion is the application version from the Info object.
	APIVersion string
	// Tags lists declared tag names in source order.
	Tags []string
	// Components contains normalized reusable definitions.
	Components Components
	// Operations contains HTTP operations in deterministic path-and-method order.
	Operations []Operation
}

// Components contains reusable OpenAPI definitions retained by the importer.
type Components struct {
	// Schemas contains reusable schema definitions sorted by source name.
	Schemas []NamedSchema
	// Parameters contains reusable parameter definitions.
	Parameters []NamedParameter
	// RequestBodies contains reusable request body definitions.
	RequestBodies []NamedRequestBody
	// Responses contains reusable response definitions.
	Responses []NamedResponse
	// Headers contains reusable header definitions.
	Headers []NamedHeader
}

// NamedSchema is a reusable schema and its deterministic Go identifier.
type NamedSchema struct {
	// Name is the authored OpenAPI component key.
	Name string
	// GoName is the deterministic collision-safe Go identifier.
	GoName string
	// Schema is the normalized component schema.
	Schema *Schema
}

// NamedParameter is a reusable parameter definition.
type NamedParameter struct {
	// Name is the authored OpenAPI component key.
	Name string
	// Parameter is the normalized component parameter.
	Parameter Parameter
}

// NamedRequestBody is a reusable request body definition.
type NamedRequestBody struct {
	// Name is the authored OpenAPI component key.
	Name string
	// RequestBody is the normalized component request body.
	RequestBody RequestBody
}

// NamedResponse is a reusable response definition.
type NamedResponse struct {
	// Name is the authored OpenAPI component key.
	Name string
	// Response is the normalized component response.
	Response Response
}

// NamedHeader is a reusable response header definition.
type NamedHeader struct {
	// Name is the authored header name or component key.
	Name string
	// Header is the normalized header definition.
	Header Header
}

// Operation is a normalized HTTP operation.
type Operation struct {
	// Method is the uppercase HTTP method.
	Method string
	// Path is the authored OpenAPI path template.
	Path string
	// OperationID is the authored OpenAPI operationId, if present.
	OperationID string
	// GoName is the deterministic collision-safe method identifier.
	GoName string
	// Summary is the authored operation summary.
	Summary string
	// Description is the authored operation description.
	Description string
	// Tags contains the operation tag names in source order.
	Tags []string
	// Deprecated reports whether the operation is deprecated.
	Deprecated bool
	// Parameters contains inherited path-item and operation parameters.
	Parameters []Parameter
	// RequestBody is the operation request body, if any.
	RequestBody *RequestBody
	// Responses contains responses sorted by status code.
	Responses []StatusResponse
}

// Parameter describes a path, query, header, or cookie parameter. Ref is set
// instead of the remaining fields for a component reference.
type Parameter struct {
	// Ref is the authored local component reference, when retained as a reference.
	Ref string
	// Name is the wire parameter name.
	Name string
	// In is path, query, header, or cookie.
	In string
	// Description is the parameter description.
	Description string
	// Required reports whether the parameter is required.
	Required bool
	// Deprecated reports whether the parameter is deprecated.
	Deprecated bool
	// AllowEmptyValue reports whether an empty parameter value is permitted.
	AllowEmptyValue bool
	// Schema describes the parameter value.
	Schema *Schema
}

// RequestBody describes a request body. Ref is set instead of the remaining
// fields for a component reference.
type RequestBody struct {
	// Ref is the authored local component reference, when retained as a reference.
	Ref string
	// Description is the request body description.
	Description string
	// Required reports whether the request body is required.
	Required bool
	// ContentType is the single request media type.
	ContentType string
	// Schema describes the request body.
	Schema *Schema
}

// StatusResponse associates an HTTP response with its status code.
type StatusResponse struct {
	// Status is a concrete three-digit HTTP status code.
	Status string
	// Response is the normalized response definition.
	Response Response
}

// Response describes an HTTP response. Ref is set instead of the remaining
// fields for a component reference.
type Response struct {
	// Ref is the authored local component reference, when retained as a reference.
	Ref string
	// Description is the response description.
	Description string
	// ContentType is the single response media type, if the response has a body.
	ContentType string
	// Schema describes the response body.
	Schema *Schema
	// Headers contains response headers sorted by wire name.
	Headers []NamedHeader
}

// Header describes a response header. Ref is set instead of the remaining
// fields for a component reference.
type Header struct {
	// Ref is the authored local component reference, when retained as a reference.
	Ref string
	// Description is the response header description.
	Description string
	// Required reports whether the response header is required.
	Required bool
	// Deprecated reports whether the response header is deprecated.
	Deprecated bool
	// Schema describes the response header value.
	Schema *Schema
}

// Schema is the normalized JSON Schema subset expressible by the first import
// phase. Ref is set instead of the remaining fields for a component reference.
type Schema struct {
	// Ref is the authored local schema component reference.
	Ref string
	// Type is the single JSON Schema type.
	Type string
	// Format is the authored schema format.
	Format string
	// Title is the schema title.
	Title string
	// Description is the schema description.
	Description string
	// Bases contains local object schemas extended by this schema.
	Bases []*Schema
	// Properties contains object properties sorted by wire name.
	Properties []NamedProperty
	// Required contains sorted required property names.
	Required []string
	// Items describes array elements.
	Items *Schema
	// AdditionalProperties describes map values or object openness.
	AdditionalProperties *AdditionalProperties
	// Enum contains decoded scalar enum values.
	Enum []any
	// Pattern is the string validation pattern.
	Pattern string
	// Minimum is the inclusive numeric lower bound.
	Minimum *float64
	// Maximum is the inclusive numeric upper bound.
	Maximum *float64
	// ExclusiveMinimum is the exclusive numeric lower bound.
	ExclusiveMinimum *float64
	// ExclusiveMaximum is the exclusive numeric upper bound.
	ExclusiveMaximum *float64
	// MinLength is the minimum string or collection length.
	MinLength *int64
	// MaxLength is the maximum string or collection length.
	MaxLength *int64
	// MinItems is the minimum array length.
	MinItems *int64
	// MaxItems is the maximum array length.
	MaxItems *int64
	// Deprecated reports whether the schema declares the JSON Schema
	// deprecated keyword.
	Deprecated bool
	// ReadOnly reports whether the schema declares the JSON Schema readOnly
	// keyword.
	ReadOnly bool
	// WriteOnly reports whether the schema declares the JSON Schema writeOnly
	// keyword.
	WriteOnly bool
	// Default holds the decoded JSON Schema default value, or nil when the
	// schema declares no default.
	Default                *SchemaDefault
	unsupportedComposition bool
}

// SchemaDefault holds a decoded JSON Schema default value together with its
// presence, distinguishing an authored default from no default at all.
type SchemaDefault struct {
	// Value is the decoded default value.
	Value any
}

// NamedProperty is an object property in deterministic source-name order.
type NamedProperty struct {
	// Name is the authored JSON property name.
	Name string
	// Schema describes the property value.
	Schema *Schema
}

// AdditionalProperties describes the boolean-or-schema JSON Schema keyword.
type AdditionalProperties struct {
	// Allowed retains an authored boolean value.
	Allowed *bool
	// Schema describes map values when additionalProperties is a schema.
	Schema *Schema
}
