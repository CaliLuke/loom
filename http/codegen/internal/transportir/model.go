package transportir

import "github.com/CaliLuke/loom/expr"

type (
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
		Payload             *expr.AttributeExpr
		Body                *expr.AttributeExpr
		RawBody             *expr.AttributeExpr
		StreamingBody       *expr.AttributeExpr
		BodyOrigin          string
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
		IDAttribute         string
		IDAttributeRequired bool
	}

	ResponseStatus struct {
		Error        *Error
		StatusCode   int
		Description  string
		ContentType  string
		ContentTypes []string
		Headers      []*Header
		Cookies      []*Cookie
		Body         *expr.AttributeExpr
		DocumentBody *expr.AttributeExpr
		BodyOrigin   string
		TagName      string
		TagValue     string
		IsError      bool
		EmitExamples bool
		IsWebSocket  bool
		BinaryBody   bool
		Meta         expr.MetaExpr
		Links        []*ResponseLink
	}

	Route struct {
		Index      int
		Method     string
		Path       string
		SourcePath string
		Wildcards  []string
	}

	Stream struct {
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
	}

	Redirect struct {
		URL        string
		StatusCode int
	}

	Parameter struct {
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
		Name             string
		HTTPName         string
		Attribute        *expr.AttributeExpr
		Required         bool
		PrimitivePointer bool
	}

	Cookie struct {
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
