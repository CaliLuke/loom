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
	// TagMetadata contains declared tag details in source order.
	TagMetadata []Tag
	// Extensions contains document-level vendor extensions.
	Extensions map[string]any
	// SecurityDefined reports whether the source declares the security member.
	SecurityDefined bool
	// Security contains root security requirement alternatives in source order.
	Security SecurityRequirements
	// Components contains normalized reusable definitions.
	Components Components
	// Operations contains HTTP operations in deterministic path-and-method order.
	Operations []Operation
}

// Tag contains OpenAPI metadata for one declared tag.
type Tag struct {
	// Name is the authored tag name.
	Name string
	// Summary is the short display label.
	Summary string
	// Description describes the tagged operations.
	Description string
	// Parent names the containing OpenAPI 3.2 tag.
	Parent string
	// Kind classifies the tag in OpenAPI 3.2.
	Kind string
	// ExternalDocsURL locates additional documentation.
	ExternalDocsURL string
	// ExternalDocsDescription describes the additional documentation.
	ExternalDocsDescription string
	// Extensions contains tag-level vendor extensions.
	Extensions map[string]any
}

// Components contains reusable OpenAPI definitions retained by the importer.
type Components struct {
	// SecuritySchemes contains supported reusable security schemes.
	SecuritySchemes []SecurityScheme
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

// SecurityScheme is a supported reusable OpenAPI security scheme.
type SecurityScheme struct {
	// Name is the authored component key.
	Name string
	// GoName is the deterministic collision-safe variable identifier.
	GoName string
	// Type is the OpenAPI security scheme type.
	Type string
	// Scheme is the HTTP authentication scheme for type http.
	Scheme string
	// Description describes the credential contract.
	Description string
	// In identifies the credential location: header, query, or cookie.
	In string
	// ParameterName is the authored wire name of the credential.
	ParameterName string
	// Deprecated reports whether use of the scheme is discouraged.
	Deprecated bool
	// OAuth2MetadataURL locates OAuth authorization server metadata.
	OAuth2MetadataURL string
	// OAuthFlows contains supported OAuth2 flows in specification order.
	OAuthFlows []OAuthFlow
	// Scopes contains the shared OAuth2 scope definitions.
	Scopes []SecurityScope
	// Extensions contains scheme-level vendor extensions.
	Extensions map[string]any
}

// OAuthFlow is one OAuth2 authorization flow.
type OAuthFlow struct {
	// Kind is authorizationCode, implicit, password, or clientCredentials.
	Kind string
	// AuthorizationURL is the flow authorization endpoint.
	AuthorizationURL string
	// TokenURL is the flow token endpoint.
	TokenURL string
	// RefreshURL is the optional flow refresh endpoint.
	RefreshURL string
}

// SecurityScope is one OAuth2 scope definition.
type SecurityScope struct {
	// Name is the authored scope value.
	Name string
	// Description explains the permission granted by the scope.
	Description string
}

// SecurityRequirements lists alternative OpenAPI security requirement objects.
type SecurityRequirements []SecurityRequirement

// SecurityRequirement lists schemes that must all succeed for one alternative.
type SecurityRequirement struct {
	// Schemes lists required schemes in source order. An empty list permits
	// anonymous access as one alternative.
	Schemes []SecurityRequirementScheme
}

// SecurityRequirementScheme references one component security scheme.
type SecurityRequirementScheme struct {
	// Name is the referenced security scheme component key.
	Name string
	// Scopes contains the authored scopes. Supported API-key schemes require
	// this list to be empty.
	Scopes []string
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
	// Extensions contains operation-level vendor extensions.
	Extensions map[string]any
	// SecurityDefined reports whether the operation overrides root security.
	SecurityDefined bool
	// Security contains operation security requirement alternatives.
	Security SecurityRequirements
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
	// Style is the authored serialization style when Loom supports it.
	Style string
	// AllowReserved permits reserved URI characters in the encoded value.
	AllowReserved bool
	// Extensions contains parameter-level vendor extensions.
	Extensions map[string]any
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
	// ContentTypes are the request media types that share Schema and Examples.
	ContentTypes []string
	// Schema describes the request body.
	Schema *Schema
	// Examples contains examples declared on the request media type.
	Examples []Example
	// Extensions contains request-body-level vendor extensions.
	Extensions map[string]any
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
	// Summary is the OpenAPI 3.2 response summary.
	Summary string
	// ContentType is the single response media type, if the response has a body.
	ContentType string
	// Schema describes the response body.
	Schema *Schema
	// Examples contains examples declared on the response media type.
	Examples []Example
	// Headers contains response headers sorted by wire name.
	Headers []NamedHeader
	// Extensions contains response-level vendor extensions.
	Extensions map[string]any
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
	// AllowReserved permits reserved URI characters in the encoded value.
	AllowReserved bool
	// Schema describes the response header value.
	Schema *Schema
}

// Schema is the normalized JSON Schema subset expressible by the first import
// phase. Ref may be accompanied by constraints authored beside a supported
// single-reference allOf wrapper.
type Schema struct {
	// Ref is the authored local schema component reference.
	Ref string
	// Unconstrained reports that the schema accepts every JSON value.
	Unconstrained bool
	// Type is the single JSON Schema type.
	Type string
	// Nullable reports that the schema also accepts the JSON null value.
	Nullable bool
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
	Default *SchemaDefault
	// Examples contains decoded examples declared by example or examples.
	Examples []Example
	// Extensions contains schema-level vendor extensions.
	Extensions             map[string]any
	unsupportedComposition bool
}

// Example is an example value retained from a schema or media type.
type Example struct {
	// Name is the authored media example key or a deterministic schema example name.
	Name string
	// Summary is the authored media example summary.
	Summary string
	// Description is the authored example description.
	Description string
	// ComponentName is the reusable OpenAPI example component name.
	ComponentName string
	// DataValue reports that Value belongs in the OpenAPI 3.2 dataValue field.
	DataValue bool
	// SerializedValue is the authored OpenAPI 3.2 serialized representation.
	SerializedValue string
	// Value is the decoded example value.
	Value any
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

type parameterOccurrence struct {
	parameter Parameter
	path      string
}
