package ir

import "encoding/json/v2"

type (
	// Document is the root OpenAPI-oriented IR document.
	Document struct {
		Paths      map[string]*PathItem
		Components *Components
	}

	// Components contains reusable IR components.
	Components struct {
		Schemas       map[string]*Schema
		Parameters    map[string]*ParameterRef
		Headers       map[string]*HeaderRef
		RequestBodies map[string]*RequestBodyRef
		Responses     map[string]*ResponseRef
		Examples      map[string]*ExampleRef
	}

	// PathItem groups operations by method.
	PathItem struct {
		Operations map[string]*Operation
	}

	// Operation describes an IR operation.
	Operation struct {
		Tags         []string
		Summary      string
		Description  string
		OperationID  string
		Parameters   []*ParameterRef
		RequestBody  *RequestBodyRef
		Responses    map[string]*ResponseRef
		Deprecated   bool
		Security     []map[string][]string
		ExternalDocs *ExternalDocs
		Extensions   map[string]any
	}

	// ParameterRef is a parameter reference or value.
	ParameterRef struct {
		Ref   string
		Value *Parameter
	}

	// Parameter describes an IR parameter object.
	Parameter struct {
		Name             string
		In               string
		ComponentName    string `json:"-"`
		Description      string
		Style            string
		Explode          *bool
		AllowEmptyValue  bool
		AllowReserved    bool
		Deprecated       bool
		Required         bool
		Schema           *Schema
		Example          any
		Examples         map[string]*ExampleRef
		Content          map[string]*MediaType
		WholeQueryString bool
		Extensions       map[string]any
	}

	// RequestBodyRef is a request body reference or value.
	RequestBodyRef struct {
		Ref   string
		Value *RequestBody
	}

	// RequestBody describes an IR request body.
	RequestBody struct {
		Description   string
		Required      bool
		ComponentName string `json:"-"`
		Content       map[string]*MediaType
		Extensions    map[string]any
	}

	// ResponseRef is a response reference or value.
	ResponseRef struct {
		Ref   string
		Value *Response
	}

	// Response describes an IR response.
	Response struct {
		Description     string
		Summary         string
		OmitDescription bool
		ComponentName   string `json:"-"`
		Headers         map[string]*HeaderRef
		Content         map[string]*MediaType
		Links           map[string]*ResponseLinkRef
		Extensions      map[string]any
	}

	// ResponseLinkRef is a response link reference or value.
	ResponseLinkRef struct {
		Ref   string
		Value *ResponseLink
	}

	// ResponseLink describes an OpenAPI response link.
	ResponseLink struct {
		OperationID  string
		OperationRef string
		Description  string
		Parameters   map[string]any
		RequestBody  any
		Extensions   map[string]any
	}

	// MediaType describes an IR media type.
	MediaType struct {
		Schema        *Schema
		Example       any
		Examples      map[string]*ExampleRef
		ComponentName string
		Metadata      map[string][]string
		Extensions    map[string]any
	}

	// HeaderRef is a header reference or value.
	HeaderRef struct {
		Ref   string
		Value *Header
	}

	// Header describes an IR header.
	Header struct {
		Description   string
		Required      bool
		AllowReserved bool
		Schema        *Schema
		Example       any
		Examples      map[string]*ExampleRef
		Extensions    map[string]any
	}

	// ExampleRef is an example reference or value.
	ExampleRef struct {
		Ref   string
		Value *Example
	}

	// Example describes an IR example object.
	Example struct {
		Summary         string
		Description     string
		ComponentName   string
		Value           any
		DataValue       any
		SerializedValue string
	}

	// NullExample is a non-nil marker that marshals as an explicit JSON or YAML
	// null while remaining distinguishable from an omitted example field.
	NullExample struct{}

	// ExternalDocs describes operation-level external documentation.
	ExternalDocs struct {
		Description string
		URL         string
	}

	// BodyTypes groups endpoint bodies and component schemas.
	BodyTypes struct {
		Services   map[string]map[string]*EndpointBodies
		Components map[string]*Schema
	}

	// EndpointBodies describes the request and response body schemas for one endpoint.
	EndpointBodies struct {
		RequestBody    *Schema
		ResponseBodies map[int][]*Schema
	}

	// Schema represents a renderer-neutral schema node.
	Schema struct {
		Ref          string
		Type         string
		Format       string
		Items        *Schema
		Properties   map[string]*Schema
		Defs         map[string]*Schema
		Title        string `json:",omitzero"`
		Description  string
		DefaultValue any
		Example      any

		ReadOnly         bool
		WriteOnly        bool
		Deprecated       bool
		ContentEncoding  string
		ContentMediaType string
		ContentSchema    *Schema

		Enum                  []any
		Pattern               string
		ExclusiveMinimum      *float64
		Minimum               *float64
		ExclusiveMaximum      *float64
		Maximum               *float64
		MinLength             *int
		MaxLength             *int
		MinItems              *int
		MaxItems              *int
		Required              []string
		AdditionalProperties  *BoolOrSchema
		UnevaluatedProperties *BoolOrSchema

		AllOf         []*Schema `json:",omitzero,omitempty"`
		AnyOf         []*Schema
		OneOf         []*Schema
		Discriminator *Discriminator
		XML           *XML

		Extensions map[string]any
	}

	// BoolOrSchema models JSON Schema bool-or-schema fields.
	BoolOrSchema struct {
		Bool   *bool
		Schema *Schema
	}

	// Discriminator describes union selection.
	Discriminator struct {
		PropertyName   string
		Mapping        map[string]string
		DefaultMapping string
		Optional       bool
	}

	// XML describes how a schema maps to XML nodes.
	XML struct {
		Name      string
		Namespace string
		Prefix    string
		NodeType  string
	}

	stableSchemaEncoding struct {
		Ref          string
		Type         string
		Format       string
		Items        *Schema
		Properties   map[string]*Schema
		Defs         map[string]*Schema
		Title        string `json:",omitzero"`
		Description  string
		DefaultValue any
		Example      any

		Media            any
		ReadOnly         bool
		WriteOnly        bool
		Deprecated       bool
		ContentEncoding  string
		ContentMediaType string
		ContentSchema    *Schema
		PathStart        string
		Links            []any

		Enum                  []any
		Pattern               string
		ExclusiveMinimum      *float64
		Minimum               *float64
		ExclusiveMaximum      *float64
		Maximum               *float64
		MinLength             *int
		MaxLength             *int
		MinItems              *int
		MaxItems              *int
		Required              []string
		AdditionalProperties  *BoolOrSchema
		UnevaluatedProperties *BoolOrSchema

		AllOf         []*Schema `json:",omitzero,omitempty"`
		AnyOf         []*Schema
		OneOf         []*Schema
		Discriminator *Discriminator
		XML           *XML

		Extensions map[string]any
	}
)

// MarshalJSON preserves the v1 structural encoding used to derive reusable
// component names. Retired JSON Hyper-Schema slots remain zero-valued in the
// hash encoding so removing them from the active model does not rename existing
// OpenAPI components.
func (s *Schema) MarshalJSON() ([]byte, error) {
	return json.Marshal(stableSchemaEncoding{
		Ref:                   s.Ref,
		Type:                  s.Type,
		Format:                s.Format,
		Items:                 s.Items,
		Properties:            s.Properties,
		Defs:                  s.Defs,
		Title:                 s.Title,
		Description:           s.Description,
		DefaultValue:          s.DefaultValue,
		Example:               s.Example,
		ReadOnly:              s.ReadOnly,
		WriteOnly:             s.WriteOnly,
		Deprecated:            s.Deprecated,
		ContentEncoding:       s.ContentEncoding,
		ContentMediaType:      s.ContentMediaType,
		ContentSchema:         s.ContentSchema,
		Enum:                  s.Enum,
		Pattern:               s.Pattern,
		ExclusiveMinimum:      s.ExclusiveMinimum,
		Minimum:               s.Minimum,
		ExclusiveMaximum:      s.ExclusiveMaximum,
		Maximum:               s.Maximum,
		MinLength:             s.MinLength,
		MaxLength:             s.MaxLength,
		MinItems:              s.MinItems,
		MaxItems:              s.MaxItems,
		Required:              s.Required,
		AdditionalProperties:  s.AdditionalProperties,
		UnevaluatedProperties: s.UnevaluatedProperties,
		AllOf:                 s.AllOf,
		AnyOf:                 s.AnyOf,
		OneOf:                 s.OneOf,
		Discriminator:         s.Discriminator,
		XML:                   s.XML,
		Extensions:            s.Extensions,
	},
		json.Deterministic(true),
		json.FormatNilMapAsNull(true),
		json.FormatNilSliceAsNull(true),
	)
}

// MarshalJSON implements json.Marshaler.
func (NullExample) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}

// MarshalYAML implements yaml.Marshaler.
func (NullExample) MarshalYAML() (any, error) {
	return nil, nil
}
