package ir

type (
	// Document is the root OpenAPI-oriented IR document.
	Document struct {
		Paths      map[string]*PathItem
		Components *Components
	}

	// Components contains reusable IR components.
	Components struct {
		Schemas map[string]*Schema
	}

	// PathItem groups operations by method.
	PathItem struct {
		Operations map[string]*Operation
	}

	// Operation describes an IR operation.
	Operation struct {
		RequestBody *RequestBody
		Responses   map[string]*Response
	}

	// RequestBody describes an IR request body.
	RequestBody struct {
		Description string
		Required    bool
		Content     map[string]*MediaType
	}

	// Response describes an IR response.
	Response struct {
		Description string
		Content     map[string]*MediaType
	}

	// MediaType describes an IR media type.
	MediaType struct {
		Schema  *Schema
		Example any
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
		Description  string
		DefaultValue any
		Example      any

		Media     *Media
		ReadOnly  bool
		PathStart string
		Links     []*Link

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

		AnyOf         []*Schema
		OneOf         []*Schema
		Discriminator *Discriminator

		Extensions map[string]any
	}

	// BoolOrSchema models JSON Schema bool-or-schema fields.
	BoolOrSchema struct {
		Bool   *bool
		Schema *Schema
	}

	// Discriminator describes union selection.
	Discriminator struct {
		PropertyName string
		Mapping      map[string]string
	}

	// Media represents JSON hyper schema media.
	Media struct {
		BinaryEncoding string
		Type           string
	}

	// Link represents JSON hyper schema link metadata.
	Link struct {
		Title        string
		Description  string
		Rel          string
		Href         string
		Method       string
		Schema       *Schema
		TargetSchema *Schema
		ResultType   string
		EncType      string
	}
)
