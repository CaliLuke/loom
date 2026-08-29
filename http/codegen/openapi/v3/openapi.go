package openapiv3

import "github.com/CaliLuke/loom/http/codegen/openapi"

type (
	// OpenAPI is a data structure that encodes the information needed to
	// generate an OpenAPI 3.1 or 3.2 specification.
	OpenAPI struct {
		OpenAPI string `json:"openapi" yaml:"openapi"` // Required
		// Self identifies this OpenAPI document in version 3.2 and later.
		Self              string                `json:"$self,omitzero" yaml:"$self,omitempty"`
		Info              *Info                 `json:"info" yaml:"info"` // Required
		JSONSchemaDialect string                `json:"jsonSchemaDialect,omitzero" yaml:"jsonSchemaDialect,omitempty"`
		Servers           []*Server             `json:"servers,omitzero,omitempty" yaml:"servers,omitempty"`
		Paths             map[string]*PathItem  `json:"paths" yaml:"paths"` // Required
		Components        *Components           `json:"components,omitempty" yaml:"components,omitempty"`
		Tags              []*openapi.Tag        `json:"tags,omitzero,omitempty" yaml:"tags,omitempty"`
		Security          []map[string][]string `json:"security,omitzero,omitempty" yaml:"security,omitempty"`
		ExternalDocs      *openapi.ExternalDocs `json:"externalDocs,omitzero" yaml:"externalDocs,omitempty"`
		Extensions        map[string]any        `json:"-" yaml:"-"`
	}

	// Info represents an OpenAPI Info object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#info-object
	Info struct {
		Title          string         `json:"title" yaml:"title"` // Required
		Summary        string         `json:"summary,omitzero" yaml:"summary,omitempty"`
		Description    string         `json:"description,omitzero" yaml:"description,omitempty"`
		TermsOfService string         `json:"termsOfService,omitzero" yaml:"termsOfService,omitempty"`
		Contact        *Contact       `json:"contact,omitzero" yaml:"contact,omitempty"`
		License        *License       `json:"license,omitzero" yaml:"license,omitempty"`
		Version        string         `json:"version" yaml:"version"` // Required
		Extensions     map[string]any `json:"-" yaml:"-"`
	}

	// Server represents an OpenAPI Server object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#serverObject
	Server struct {
		// Name identifies the server in OpenAPI 3.2 and later.
		Name        string                     `json:"name,omitzero" yaml:"name,omitempty"`
		URL         string                     `json:"url" yaml:"url"`
		Description string                     `json:"description,omitzero" yaml:"description,omitempty"`
		Variables   map[string]*ServerVariable `json:"variables,omitzero,omitempty" yaml:"variables,omitempty"`
	}

	// PathItem represents an OpenAPI Path Item object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#pathItemObject
	PathItem struct {
		Ref         string     `json:"$ref,omitzero" yaml:"$ref,omitempty"`
		Summary     string     `json:"summary,omitzero" yaml:"summary,omitempty"`
		Description string     `json:"description,omitzero" yaml:"description,omitempty"`
		Connect     *Operation `json:"connect,omitzero" yaml:"connect,omitempty"`
		Delete      *Operation `json:"delete,omitzero" yaml:"delete,omitempty"`
		Get         *Operation `json:"get,omitzero" yaml:"get,omitempty"`
		Head        *Operation `json:"head,omitzero" yaml:"head,omitempty"`
		Options     *Operation `json:"options,omitzero" yaml:"options,omitempty"`
		Patch       *Operation `json:"patch,omitzero" yaml:"patch,omitempty"`
		Post        *Operation `json:"post,omitzero" yaml:"post,omitempty"`
		Put         *Operation `json:"put,omitzero" yaml:"put,omitempty"`
		Trace       *Operation `json:"trace,omitzero" yaml:"trace,omitempty"`
		// Query describes the QUERY operation in OpenAPI 3.2 and later.
		Query *Operation `json:"query,omitzero" yaml:"query,omitempty"`
		// AdditionalOperations describes operations for additional HTTP methods in OpenAPI 3.2 and later.
		AdditionalOperations map[string]*Operation `json:"additionalOperations,omitzero,omitempty" yaml:"additionalOperations,omitempty"`
		Servers              []*Server             `json:"servers,omitzero,omitempty" yaml:"servers,omitempty"`
		Parameters           []*ParameterRef       `json:"parameters,omitzero,omitempty" yaml:"parameters,omitempty"`
		Extensions           map[string]any        `json:"-" yaml:"-"`
	}

	// Components represents an OpenAPI Components object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#componentsObject
	Components struct {
		Schemas         map[string]*openapi.Schema    `json:"schemas,omitzero,omitempty" yaml:"schemas,omitempty"`
		Parameters      map[string]*ParameterRef      `json:"parameters,omitzero,omitempty" yaml:"parameters,omitempty"`
		Headers         map[string]*HeaderRef         `json:"headers,omitzero,omitempty" yaml:"headers,omitempty"`
		RequestBodies   map[string]*RequestBodyRef    `json:"requestBodies,omitzero,omitempty" yaml:"requestBodies,omitempty"`
		Responses       map[string]*ResponseRef       `json:"responses,omitzero,omitempty" yaml:"responses,omitempty"`
		SecuritySchemes map[string]*SecuritySchemeRef `json:"securitySchemes,omitzero,omitempty" yaml:"securitySchemes,omitempty"`
		Examples        map[string]*ExampleRef        `json:"examples,omitzero,omitempty" yaml:"examples,omitempty"`
		Links           map[string]*LinkRef           `json:"links,omitzero,omitempty" yaml:"links,omitempty"`
		Callbacks       map[string]*CallbackRef       `json:"callbacks,omitzero,omitempty" yaml:"callbacks,omitempty"`
		// MediaTypes contains reusable Media Type Objects in OpenAPI 3.2 and later.
		MediaTypes map[string]*MediaTypeRef `json:"mediaTypes,omitzero,omitempty" yaml:"mediaTypes,omitempty"`
		Extensions map[string]any           `json:"-" yaml:"-"`
	}

	// Contact represents an OpenAPI Contact object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#contactObject
	Contact struct {
		Name       string         `json:"name,omitzero" yaml:"name,omitempty"`
		URL        string         `json:"url,omitzero" yaml:"url,omitempty"`
		Email      string         `json:"email,omitzero" yaml:"email,omitempty"`
		Extensions map[string]any `json:"-" yaml:"-"`
	}

	// License represents an OpenAPI License object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.1.md#license-object
	License struct {
		Name       string         `json:"name" yaml:"name"` // Required
		Identifier string         `json:"identifier,omitzero" yaml:"identifier,omitempty"`
		URL        string         `json:"url,omitzero" yaml:"url,omitempty"`
		Extensions map[string]any `json:"-" yaml:"-"`
	}

	// ServerVariable represents an OpenAPI Server Variable object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#serverVariableObject
	ServerVariable struct {
		Enum        []any  `json:"enum,omitzero,omitempty" yaml:"enum,omitempty"`
		Default     any    `json:"default,omitzero" yaml:"default,omitempty"`
		Description string `json:"description,omitzero" yaml:"description,omitempty"`
	}

	// Operation represents an OpenAPI Operation object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#operationObject
	Operation struct {
		Tags         []string                `json:"tags,omitzero,omitempty" yaml:"tags,omitempty"`
		Summary      string                  `json:"summary,omitzero" yaml:"summary,omitempty"`
		Description  string                  `json:"description,omitzero" yaml:"description,omitempty"`
		OperationID  string                  `json:"operationId,omitzero" yaml:"operationId,omitempty"`
		Parameters   []*ParameterRef         `json:"parameters,omitzero,omitempty" yaml:"parameters,omitempty"`
		RequestBody  *RequestBodyRef         `json:"requestBody,omitzero" yaml:"requestBody,omitempty"`
		Responses    map[string]*ResponseRef `json:"responses" yaml:"responses"` // Required
		Callbacks    map[string]*CallbackRef `json:"callbacks,omitzero,omitempty" yaml:"callbacks,omitempty"`
		Deprecated   bool                    `json:"deprecated,omitzero" yaml:"deprecated,omitempty"`
		Security     []map[string][]string   `json:"security,omitzero,omitempty" yaml:"security,omitempty"`
		Servers      []*Server               `json:"servers,omitzero,omitempty" yaml:"servers,omitempty"`
		ExternalDocs *openapi.ExternalDocs   `json:"externalDocs,omitzero" yaml:"externalDocs,omitempty"`
		Extensions   map[string]any          `json:"-" yaml:"-"`
	}

	// Parameter represents an OpenAPI Parameter object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#parameterObject
	Parameter struct {
		Name            string                 `json:"name,omitzero" yaml:"name,omitempty"`
		In              string                 `json:"in,omitzero" yaml:"in,omitempty"`
		Description     string                 `json:"description,omitzero" yaml:"description,omitempty"`
		Style           string                 `json:"style,omitzero" yaml:"style,omitempty"`
		Explode         *bool                  `json:"explode,omitzero" yaml:"explode,omitempty"`
		AllowEmptyValue bool                   `json:"allowEmptyValue,omitzero" yaml:"allowEmptyValue,omitempty"`
		AllowReserved   bool                   `json:"allowReserved,omitzero" yaml:"allowReserved,omitempty"`
		Deprecated      bool                   `json:"deprecated,omitzero" yaml:"deprecated,omitempty"`
		Required        bool                   `json:"required,omitzero" yaml:"required,omitempty"`
		Schema          *openapi.Schema        `json:"schema,omitzero" yaml:"schema,omitempty"`
		Example         any                    `json:"example,omitzero" yaml:"example,omitempty"`
		Examples        map[string]*ExampleRef `json:"examples,omitzero,omitempty" yaml:"examples,omitempty"`
		Content         map[string]*MediaType  `json:"content,omitzero,omitempty" yaml:"content,omitempty"`
		// WholeQueryString marks parameters promoted to OpenAPI 3.2 querystring parameters.
		WholeQueryString bool           `json:"-" yaml:"-"`
		Extensions       map[string]any `json:"-" yaml:"-"`
	}

	// Response represents an OpenAPI Response object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#responseObject
	Response struct {
		Summary     string                `json:"summary,omitzero" yaml:"summary,omitempty"`
		Description *string               `json:"description,omitzero" yaml:"description,omitempty"`
		Headers     map[string]*HeaderRef `json:"headers,omitzero,omitempty" yaml:"headers,omitempty"`
		Content     map[string]*MediaType `json:"content,omitzero,omitempty" yaml:"content,omitempty"`
		Links       map[string]*LinkRef   `json:"links,omitzero,omitempty" yaml:"links,omitempty"`
		Extensions  map[string]any        `json:"-" yaml:"-"`
		// CompatibilityDescription preserves the generated description for a 3.1 target.
		CompatibilityDescription string `json:"-" yaml:"-"`
	}

	// MediaType represents an OpenAPI Media Type object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#mediaTypeObject
	MediaType struct {
		// Ref references a reusable Media Type Object in OpenAPI 3.2 and later.
		Ref    string          `json:"$ref,omitzero" yaml:"$ref,omitempty"`
		Schema *openapi.Schema `json:"schema,omitzero" yaml:"schema,omitempty"`
		// ItemSchema describes each item in a streaming media type in OpenAPI 3.2 and later.
		ItemSchema *openapi.Schema `json:"itemSchema,omitzero" yaml:"itemSchema,omitempty"`
		// PrefixEncoding describes encodings for tuple prefixes in OpenAPI 3.2 and later.
		PrefixEncoding []*Encoding `json:"prefixEncoding,omitzero,omitempty" yaml:"prefixEncoding,omitempty"`
		// ItemEncoding describes the encoding applied to each remaining item in OpenAPI 3.2 and later.
		ItemEncoding *Encoding              `json:"itemEncoding,omitzero" yaml:"itemEncoding,omitempty"`
		Example      any                    `json:"example,omitzero" yaml:"example,omitempty"`
		Examples     map[string]*ExampleRef `json:"examples,omitzero,omitempty" yaml:"examples,omitempty"`
		Encoding     map[string]*Encoding   `json:"encoding,omitzero,omitempty" yaml:"encoding,omitempty"`
		Extensions   map[string]any         `json:"-" yaml:"-"`
		// ComponentName names a reusable OpenAPI 3.2 media type component.
		ComponentName string `json:"-" yaml:"-"`
		// UseItemSchema marks sequential media whose schema applies to each streamed item.
		UseItemSchema bool `json:"-" yaml:"-"`
	}

	// Encoding represents an OpenAPI Encoding object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#encodingObject
	Encoding struct {
		// ContentType identifies the media type used to encode the value.
		ContentType string `json:"contentType,omitzero" yaml:"contentType,omitempty"`
		// Headers describes additional headers associated with the encoded value.
		Headers map[string]*HeaderRef `json:"headers,omitzero,omitempty" yaml:"headers,omitempty"`
		// Style describes how the encoded value is serialized.
		Style string `json:"style,omitzero" yaml:"style,omitempty"`
		// Explode controls whether arrays and objects produce separate parameters.
		Explode *bool `json:"explode,omitzero" yaml:"explode,omitempty"`
		// AllowReserved permits reserved URI characters in the encoded value.
		AllowReserved bool `json:"allowReserved,omitzero" yaml:"allowReserved,omitempty"`
		// Encoding describes nested object-property encodings in OpenAPI 3.2 and later.
		Encoding map[string]*Encoding `json:"encoding,omitzero,omitempty" yaml:"encoding,omitempty"`
		// PrefixEncoding describes nested tuple-prefix encodings in OpenAPI 3.2 and later.
		PrefixEncoding []*Encoding `json:"prefixEncoding,omitzero,omitempty" yaml:"prefixEncoding,omitempty"`
		// ItemEncoding describes the nested encoding for remaining array items in OpenAPI 3.2 and later.
		ItemEncoding *Encoding `json:"itemEncoding,omitzero" yaml:"itemEncoding,omitempty"`
		// Extensions contains specification extensions keyed by x-prefixed names.
		Extensions map[string]any `json:"-" yaml:"-"`
	}

	// Header represents an OpenAPI Header object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#headerObject
	Header struct {
		Description     string                 `json:"description,omitzero" yaml:"description,omitempty"`
		Style           string                 `json:"style,omitzero" yaml:"style,omitempty"`
		Explode         *bool                  `json:"explode,omitzero" yaml:"explode,omitempty"`
		AllowEmptyValue bool                   `json:"allowEmptyValue,omitzero" yaml:"allowEmptyValue,omitempty"`
		AllowReserved   bool                   `json:"allowReserved,omitzero" yaml:"allowReserved,omitempty"`
		Deprecated      bool                   `json:"deprecated,omitzero" yaml:"deprecated,omitempty"`
		Required        bool                   `json:"required,omitzero" yaml:"required,omitempty"`
		Schema          *openapi.Schema        `json:"schema,omitzero" yaml:"schema,omitempty"`
		Example         any                    `json:"example,omitzero" yaml:"example,omitempty"`
		Examples        map[string]*ExampleRef `json:"examples,omitzero,omitempty" yaml:"examples,omitempty"`
		Content         map[string]*MediaType  `json:"content,omitzero,omitempty" yaml:"content,omitempty"`
		Extensions      map[string]any         `json:"-" yaml:"-"`
	}

	// Link represents an OpenAPI Link object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#linkObject
	Link struct {
		OperationID  string         `json:"operationId,omitzero" yaml:"operationId,omitempty"`
		OperationRef string         `json:"operationRef,omitzero" yaml:"operationRef,omitempty"`
		Description  string         `json:"description,omitzero" yaml:"description,omitempty"`
		Parameters   map[string]any `json:"parameters,omitzero,omitempty" yaml:"parameters,omitempty"`
		Server       *Server        `json:"server,omitzero" yaml:"server,omitempty"`
		RequestBody  any            `json:"requestBody,omitzero" yaml:"requestBody,omitempty"`
		Extensions   map[string]any `json:"-" yaml:"-"`
	}

	// Example represents an OpenAPI Example object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#exampleObject
	Example struct {
		Summary     string `json:"summary,omitzero" yaml:"summary,omitempty"`
		Description string `json:"description,omitzero" yaml:"description,omitempty"`
		Value       any    `json:"value,omitzero" yaml:"value,omitempty"`
		// DataValue contains the example's parsed data value in OpenAPI 3.2 and later.
		DataValue any `json:"dataValue,omitzero" yaml:"dataValue,omitempty"`
		// SerializedValue contains the example's serialized representation in OpenAPI 3.2 and later.
		SerializedValue string         `json:"serializedValue,omitzero" yaml:"serializedValue,omitempty"`
		ExternalValue   string         `json:"externalValue,omitzero" yaml:"externalValue,omitempty"`
		Extensions      map[string]any `json:"-" yaml:"-"`
		// CompatibilityValue preserves the ordinary example value for OpenAPI 3.1 output.
		CompatibilityValue any `json:"-" yaml:"-"`
	}

	// RequestBody represents an OpenAPI RequestBody object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#requestBodyObject
	RequestBody struct {
		Description string                `json:"description,omitzero" yaml:"description,omitempty"`
		Required    bool                  `json:"required,omitzero" yaml:"required,omitempty"`
		Content     map[string]*MediaType `json:"content,omitzero,omitempty" yaml:"content,omitempty"`
		Extensions  map[string]any        `json:"-" yaml:"-"`
	}

	// SecurityScheme represents an OpenAPI SecurityScheme object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#securitySchemeObject
	SecurityScheme struct {
		Type         string      `json:"type,omitzero" yaml:"type,omitempty"`
		Description  string      `json:"description,omitzero" yaml:"description,omitempty"`
		Name         string      `json:"name,omitzero" yaml:"name,omitempty"`
		In           string      `json:"in,omitzero" yaml:"in,omitempty"`
		Scheme       string      `json:"scheme,omitzero" yaml:"scheme,omitempty"`
		BearerFormat string      `json:"bearerFormat,omitzero" yaml:"bearerFormat,omitempty"`
		Flows        *OAuthFlows `json:"flows,omitzero" yaml:"flows,omitempty"`
		// OAuth2MetadataURL locates OAuth 2.0 authorization server metadata in OpenAPI 3.2 and later.
		OAuth2MetadataURL string `json:"oauth2MetadataUrl,omitzero" yaml:"oauth2MetadataUrl,omitempty"`
		// Deprecated reports whether use of the security scheme is discouraged.
		Deprecated bool           `json:"deprecated,omitzero" yaml:"deprecated,omitempty"`
		Extensions map[string]any `json:"-" yaml:"-"`
	}

	// OAuthFlows represents an OpenAPI OAuthFlows object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#oauthFlowsObject
	OAuthFlows struct {
		Implicit          *OAuthFlow `json:"implicit,omitzero" yaml:"implicit,omitempty"`
		Password          *OAuthFlow `json:"password,omitzero" yaml:"password,omitempty"`
		ClientCredentials *OAuthFlow `json:"clientCredentials,omitzero" yaml:"clientCredentials,omitempty"`
		AuthorizationCode *OAuthFlow `json:"authorizationCode,omitzero" yaml:"authorizationCode,omitempty"`
		// DeviceAuthorization describes the OAuth device authorization flow in OpenAPI 3.2 and later.
		DeviceAuthorization *OAuthFlow     `json:"deviceAuthorization,omitzero" yaml:"deviceAuthorization,omitempty"`
		Extensions          map[string]any `json:"-" yaml:"-"`
	}

	// OAuthFlow represents an OpenAPI OAuthFlow object as defined in
	// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#oauthFlowObject
	OAuthFlow struct {
		AuthorizationURL string `json:"authorizationUrl,omitzero" yaml:"authorizationUrl,omitempty"`
		TokenURL         string `json:"tokenUrl,omitzero" yaml:"tokenUrl,omitempty"`
		RefreshURL       string `json:"refreshUrl,omitzero" yaml:"refreshUrl,omitempty"`
		// DeviceAuthorizationURL locates the OAuth device authorization endpoint in OpenAPI 3.2 and later.
		DeviceAuthorizationURL string            `json:"deviceAuthorizationUrl,omitzero" yaml:"deviceAuthorizationUrl,omitempty"`
		Scopes                 map[string]string `json:"scopes" yaml:"scopes"`
		Extensions             map[string]any    `json:"-" yaml:"-"`
	}

	// These types are used in openapi.MarshalJSON() to avoid recursive call of json.Marshal().
	_Info           Info
	_PathItem       PathItem
	_Operation      Operation
	_Parameter      Parameter
	_Response       Response
	_SecurityScheme SecurityScheme
)

type operationWithSecurity struct {
	Tags         []string                `json:"tags,omitzero,omitempty" yaml:"tags,omitempty"`
	Summary      string                  `json:"summary,omitzero" yaml:"summary,omitempty"`
	Description  string                  `json:"description,omitzero" yaml:"description,omitempty"`
	OperationID  string                  `json:"operationId,omitzero" yaml:"operationId,omitempty"`
	Parameters   []*ParameterRef         `json:"parameters,omitzero,omitempty" yaml:"parameters,omitempty"`
	RequestBody  *RequestBodyRef         `json:"requestBody,omitzero" yaml:"requestBody,omitempty"`
	Responses    map[string]*ResponseRef `json:"responses" yaml:"responses"`
	Callbacks    map[string]*CallbackRef `json:"callbacks,omitzero,omitempty" yaml:"callbacks,omitempty"`
	Deprecated   bool                    `json:"deprecated,omitzero" yaml:"deprecated,omitempty"`
	Security     []map[string][]string   `json:"security" yaml:"security"`
	Servers      []*Server               `json:"servers,omitzero,omitempty" yaml:"servers,omitempty"`
	ExternalDocs *openapi.ExternalDocs   `json:"externalDocs,omitzero" yaml:"externalDocs,omitempty"`
}

// MediaType implements exampler
func (m *MediaType) setExample(val any)                     { m.Example = val }
func (m *MediaType) setExamples(val map[string]*ExampleRef) { m.Examples = val }

// Header implements exampler
func (h *Header) setExample(val any)                     { h.Example = val }
func (h *Header) setExamples(val map[string]*ExampleRef) { h.Examples = val }

// Parameter implements exampler
func (p *Parameter) setExample(val any)                     { p.Example = val }
func (p *Parameter) setExamples(val map[string]*ExampleRef) { p.Examples = val }

// MarshalJSON returns the JSON encoding of i.
func (i Info) MarshalJSON() ([]byte, error) {
	return openapi.MarshalJSON(_Info(i), i.Extensions)
}

// MarshalJSON returns the JSON encoding of p.
func (p PathItem) MarshalJSON() ([]byte, error) {
	return openapi.MarshalJSON(_PathItem(p), p.Extensions)
}

// MarshalJSON returns the JSON encoding of o.
func (o Operation) MarshalJSON() ([]byte, error) {
	if o.Security == nil {
		return openapi.MarshalJSON(_Operation(o), o.Extensions)
	}
	return openapi.MarshalJSON(o.securityView(), o.Extensions)
}

// MarshalJSON returns the JSON encoding of p.
func (p *Parameter) MarshalJSON() ([]byte, error) {
	return openapi.MarshalJSON((*_Parameter)(p), p.Extensions)
}

// MarshalJSON returns the JSON encoding of r.
func (r Response) MarshalJSON() ([]byte, error) {
	return openapi.MarshalJSON(_Response(r), r.Extensions)
}

// MarshalJSON returns the JSON encoding of s.
func (s SecurityScheme) MarshalJSON() ([]byte, error) {
	return openapi.MarshalJSON(_SecurityScheme(s), s.Extensions)
}

// MarshalYAML returns value which marshaled in place of the original value
func (i Info) MarshalYAML() (any, error) {
	return openapi.MarshalYAML(_Info(i), i.Extensions)
}

// MarshalYAML returns value which marshaled in place of the original value
func (p PathItem) MarshalYAML() (any, error) {
	return openapi.MarshalYAML(_PathItem(p), p.Extensions)
}

// MarshalYAML returns value which marshaled in place of the original value
func (o Operation) MarshalYAML() (any, error) {
	if o.Security == nil {
		return openapi.MarshalYAML(_Operation(o), o.Extensions)
	}
	return openapi.MarshalYAML(o.securityView(), o.Extensions)
}

// MarshalYAML returns value which marshaled in place of the original value
func (p *Parameter) MarshalYAML() (any, error) {
	return openapi.MarshalYAML((*_Parameter)(p), p.Extensions)
}

// MarshalYAML returns value which marshaled in place of the original value
func (r Response) MarshalYAML() (any, error) {
	return openapi.MarshalYAML(_Response(r), r.Extensions)
}

// MarshalYAML returns value which marshaled in place of the original value
func (s SecurityScheme) MarshalYAML() (any, error) {
	return openapi.MarshalYAML(_SecurityScheme(s), s.Extensions)
}

func (o Operation) securityView() operationWithSecurity {
	return operationWithSecurity{
		Tags:         o.Tags,
		Summary:      o.Summary,
		Description:  o.Description,
		OperationID:  o.OperationID,
		Parameters:   o.Parameters,
		RequestBody:  o.RequestBody,
		Responses:    o.Responses,
		Callbacks:    o.Callbacks,
		Deprecated:   o.Deprecated,
		Security:     o.Security,
		Servers:      o.Servers,
		ExternalDocs: o.ExternalDocs,
	}
}
