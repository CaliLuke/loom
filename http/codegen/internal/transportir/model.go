package transportir

import (
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

type (
	// ValueTarget retains source provenance and the actual target codec. A
	// boundary or error is explicit; an absent plan never implies validity.
	ValueTarget struct {
		// Source is the shared effective service occurrence and selected result.
		Source *service.ValueData
		// Selection is an authored member path within Source.
		Selection []string
		// Codec is selected by actual transport behavior, never media labels.
		Codec expr.ValueCodec
		// Plan is populated only after actual emitter policies are available.
		Plan expr.ValuePlan
		// Error retains a plan construction failure for consuming analysis.
		Error error
		// Boundary records an external codec whose guarantees are not built-in JSON.
		Boundary string
		// Documentary excludes runtime decoding claims for a documentation-only target.
		Documentary bool
	}

	Service struct {
		Name            string
		Meta            expr.MetaExpr
		ServiceMeta     expr.MetaExpr
		Generate        bool
		ServiceGenerate bool
		Endpoints       []*Endpoint
	}

	Endpoint struct {
		Service        *Service
		Name           string
		MethodName     string
		Description    string
		Meta           expr.MetaExpr
		MethodMeta     expr.MetaExpr
		MethodDocs     *expr.DocsExpr
		Generate       bool
		MethodGenerate bool
		IsJSONRPC      bool
		Request        *Request
		Response       *Response
		Routes         []*Route
		Stream         *Stream
		Redirect       *Redirect
		Security       *Security
	}

	Request struct {
		Payload *expr.AttributeExpr
		// BodyValue shares service source authority for the request body.
		BodyValue *ValueTarget
		// DocumentValue has documentation-only source authority when explicitly authored.
		DocumentValue *ValueTarget
		// StreamingValue shares the service streaming-payload occurrence.
		StreamingValue *ValueTarget
		Body           *expr.AttributeExpr
		// DocumentBody is the documentation-only request body schema.
		DocumentBody *expr.AttributeExpr
		// DocumentContentTypes are the documentation-only request media types.
		DocumentContentTypes []string
		// DocumentRequired is the documentation-only request body requiredness.
		DocumentRequired bool
		// StreamingBody is the body of the WebSocket messages of a streaming
		// payload, normalized like Body: named non-object types, including
		// named unions, are replaced with their underlying types.
		StreamingBody *expr.AttributeExpr
		// BodyOrigin is the attribute name of the payload attribute selected
		// with Body(name), or empty when the body is not one attribute.
		BodyOrigin string
		// BodyOriginKey is the object key of the payload attribute that
		// BodyOrigin names, such as "v:x" for "v". Requiredness, defaults
		// and pointer semantics of the attribute are looked up by this key.
		BodyOriginKey       string
		PathParams          []*Parameter
		QueryParams         []*Parameter
		Headers             []*Parameter
		Cookies             []*Parameter
		MapQueryParams      *string
		Multipart           bool
		FormEncoded         bool
		OptionalBody        bool
		MustHaveBody        bool
		SkipBodyEncode      bool
		IDAttribute         string
		IDAttributeRequired bool
	}

	Response struct {
		Result              *expr.AttributeExpr
		StreamingResult     *expr.AttributeExpr
		Responses           []*ResponseStatus
		ErrorResponses      []*ResponseStatus
		HasMixedResults     bool
		SkipBodyEncode      bool
		FileResponse        bool
		IDAttribute         string
		IDAttributeRequired bool
	}

	ResponseStatus struct {
		// BodyValue carries the effective result or error occurrence.
		BodyValue *ValueTarget
		// DocumentValue carries the documentation body authority.
		DocumentValue *ValueTarget
		// IndependentDocumentBody marks an authored OpenAPIBody contract.
		IndependentDocumentBody bool
		Error                   *Error
		StatusCode              int
		Description             string
		ContentType             string
		ContentTypes            []string
		Headers                 []*Header
		Cookies                 []*Cookie
		Body                    *expr.AttributeExpr
		DocumentBody            *expr.AttributeExpr
		BodyOrigin              string
		TagName                 string
		TagValue                string
		IsError                 bool
		EmitExamples            bool
		IsWebSocket             bool
		BinaryBody              bool
		Meta                    expr.MetaExpr
		Links                   []*ResponseLink
	}

	Route struct {
		Index      int
		Method     string
		Path       string
		SourcePath string
		Wildcards  []string
	}

	Stream struct {
		// RequestValue and ResponseValue carry stream message source authority.
		RequestValue     *ValueTarget
		ResponseValue    *ValueTarget
		Kind             expr.StreamKind
		Direction        string
		IsStreaming      bool
		Transport        string
		IsSSE            bool
		IsWebSocket      bool
		HasMixedResults  bool
		RequestHasBody   bool
		RequestPayload   *expr.AttributeExpr
		RequestMessage   *expr.AttributeExpr
		ResponseMessage  *expr.AttributeExpr
		HandshakeMethod  string
		HandshakeStatus  int
		HandshakeContent string
		SSE              *SSE
	}

	SSE struct {
		RequestIDField     string
		RequestIDPointer   bool
		NotificationMethod string
		DataField          string
		IDField            string
		EventField         string
		RetryField         string
		Projections        []*expr.SSEProjectionExpr
	}

	Redirect struct {
		URL        string
		StatusCode int
	}

	Parameter struct {
		// Value shares the selected service member and its actual text codec.
		Value            *ValueTarget
		Name             string
		HTTPName         string
		In               string
		Attribute        *expr.AttributeExpr
		Required         bool
		PrimitivePointer bool
		Map              bool
		MapQueryParams   *string
		StringSlice      bool
		Slice            bool
	}

	Security struct {
		Requirements []*expr.SecurityExpr
		Parameters   []*SecurityParameter
		Disabled     bool
	}

	SecurityParameter struct {
		Name       string
		In         string
		SchemeName string
	}

	Error struct {
		Name      string
		Attribute *expr.AttributeExpr
		Type      expr.DataType
		Remedy    *ErrorRemedy
	}

	ErrorRemedy struct {
		Code        string
		SafeMessage string
		RetryHint   string
	}

	Header struct {
		// Value shares the selected service member and its actual text codec.
		Value            *ValueTarget
		Name             string
		HTTPName         string
		Attribute        *expr.AttributeExpr
		Required         bool
		PrimitivePointer bool
	}

	Cookie struct {
		// Value shares the selected service member and its actual text codec.
		Value            *ValueTarget
		Name             string
		HTTPName         string
		Attribute        *expr.AttributeExpr
		Required         bool
		PrimitivePointer bool
		Path             string
		Domain           string
		MaxAge           string
		Secure           bool
		HTTPOnly         bool
		SameSite         expr.CookieSameSiteValue
	}

	ResponseLink struct {
		Name         string
		Operation    string
		OperationRef string
		Description  string
		RequestBody  string
		Parameters   map[string]string
	}
)
