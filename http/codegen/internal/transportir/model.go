package transportir

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

type (
	// ValueTarget retains source provenance and the actual target codec. A
	// boundary or error is explicit; an absent plan never implies validity.
	ValueTarget struct {
		// Source is the shared effective service occurrence and selected result.
		Source *service.ValueData
		// Anchor retains the original service occurrence when a transport-local
		// authored group owns Source.
		Anchor *service.ValueData
		// Selection is an authored member path within Source.
		Selection []string
		// Codec is selected by actual transport behavior, never media labels.
		Codec expr.ValueCodec
		// SSEDataField identifies the physical field passed to EncodeSSEData.
		// Its codec is derived from the emitted type and presence layout.
		SSEDataField string
		// ProjectionContext is the captured body layout used to build Plan.
		// Its immutable service registry is shared; local override maps are
		// detached from mutable generation registries.
		// Consumers treat the snapshot as read-only; rebuilding replaces it.
		ProjectionContext *codegen.AttributeContext
		// Plan is populated only after actual emitter policies are available.
		Plan expr.ValuePlan
		// AnchorPlan projects values owned by Anchor through this target.
		AnchorPlan expr.ValuePlan
		// ExampleOccurrence is the independently captured effective target used
		// for schema annotations and transport-local authored examples. Runtime
		// targets without a local authored root example leave it zero.
		ExampleOccurrence expr.ValueOccurrence
		// ExamplePlan projects values owned by ExampleOccurrence through this target.
		ExamplePlan expr.ValuePlan
		// Examples retains the selected authored group in declaration order.
		Examples []ValueExample
		// Representative retains the last entry of Examples when an authored
		// group applies; otherwise it retains the selected or synthesized result.
		Representative *ValueExample
		// ExamplesPrepared reports that the explicit preparation stage ran, even
		// when suppression or projection failure leaves no emitted value.
		ExamplesPrepared bool
		// ExampleSets binds each schema-plan node to the examples prepared for
		// that exact documentation occurrence. Records retain their own semantic
		// owners and projection plans; schema traversal never reselects a source.
		ExampleSets map[expr.ValuePlanNode]*ValueExampleSet
		// AnnotationsPrepared reports successful preparation independently of optional examples.
		AnnotationsPrepared bool
		// Annotations retains checked required values for each exact schema node.
		Annotations map[expr.ValuePlanNode]expr.JSONSchemaAnnotations
		// Error retains a plan construction failure for consuming analysis.
		Error error
		// Boundary records an external codec whose guarantees are not built-in JSON.
		Boundary string
		// Documentary excludes runtime decoding claims for a documentation-only target.
		Documentary bool
	}

	// ValueExample is one semantic example and its exact target association.
	// Authored remains opaque; consumers project Source.Example through Plan.
	ValueExample struct {
		// Source owns the occurrence and resolved or synthesized result.
		Source *service.ValueData
		// Authored is the opaque authored source, or zero for synthesis.
		Authored expr.ValueSource
		// Plan is the matching target representation for Source.Occurrence.
		Plan expr.ValuePlan
		// Summary is detached authored metadata.
		Summary string
		// Description is detached authored metadata.
		Description string
		// Meta is detached authored metadata.
		Meta expr.MetaExpr
	}

	// ValueExampleSet is the complete prepared example state for one schema
	// occurrence. An empty prepared set suppresses consumer-side fallback.
	ValueExampleSet struct {
		// Examples retains an authored group in declaration order.
		Examples []ValueExample
		// Representative is the selected authored entry or synthesized value.
		Representative *ValueExample
		// Prepared distinguishes an empty or suppressed set from missing analysis.
		Prepared bool
	}

	Service struct {
		Name            string
		Meta            expr.MetaExpr
		ServiceMeta     expr.MetaExpr
		Generate        bool
		ServiceGenerate bool
		Endpoints       []*Endpoint
		// SchemaTypeLayouts is the immutable service registry shared by schema
		// targets, including subsequently projected stream messages.
		SchemaTypeLayouts *codegen.TypeLayouts
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
		// MediaVariants retains each typed request representation independently.
		MediaVariants []*MediaVariant
		// ContentType is the admitted authored typed request media, empty when implicit.
		ContentType string
		// JSONRPCID retains the canonical envelope binding.
		JSONRPCID *expr.JSONRPCIDBinding
		Payload   *expr.AttributeExpr
		// BodyValue shares service source authority for the request body.
		BodyValue *ValueTarget
		// DocumentValue has documentation-only source authority when explicitly authored.
		DocumentValue *ValueTarget
		// DocumentValues records separately declared documentation media representations.
		DocumentValues map[string]*ValueTarget
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
		BodyOriginKey  string
		PathParams     []*Parameter
		QueryParams    []*Parameter
		Headers        []*Parameter
		Cookies        []*Parameter
		MapQueryParams *string
		Multipart      bool
		// MultipartEncoding retains the complete occurrence-owned multipart contract.
		MultipartEncoding *expr.HTTPMultipartEncoding
		FormEncoded       bool
		// Form retains the complete occurrence-owned form body encoding.
		Form           *expr.HTTPFormEncoding
		OptionalBody   bool
		MustHaveBody   bool
		SkipBodyEncode bool
	}

	Response struct {
		Result          *expr.AttributeExpr
		StreamingResult *expr.AttributeExpr
		// Responses retains validated success occurrences in selection order:
		// tagged branches precede untagged branches; status-selected branches retain
		// declaration order and are selected by the captured selector keys.
		Responses      []*ResponseStatus
		ErrorResponses []*ResponseStatus
		// Informational retains every declaration, independent of observer registration.
		Informational []*InformationalResponse
		// FinalKeys is the detached complete final-selector inventory.
		FinalKeys []expr.HTTPStatusSelector
		// InformationalKeys is the detached observation-selector inventory, not emission domains.
		InformationalKeys []expr.HTTPStatusSelector
		// StatusSelected selects successes through the native status field.
		StatusSelected bool
		// NoSuccess records an explicitly error-only endpoint.
		NoSuccess           bool
		HasMixedResults     bool
		SkipBodyEncode      bool
		FileResponse        bool
		IDAttribute         string
		IDAttributeRequired bool
	}

	// InformationalResponse names one header-only response occurrence.
	InformationalResponse struct {
		// Name is the authored declaration name used for typed sender/observer APIs.
		Name string
		// Response is the common carrier, binding and selector record.
		Response *ResponseStatus
	}

	// StatusBinding identifies the actual native field that stores an HTTP status.
	StatusBinding struct {
		// Name is the resolved native object key, including its mapped suffix.
		Name string
		// Attribute is the native field, not a projected wire-body field.
		Attribute *expr.AttributeExpr
		// Value retains the field's effective semantic occurrence when prepared.
		Value *ValueTarget
	}

	// ResponseBody retains one physical body representation and its captured
	// source authority, without duplicating response status or metadata ownership.
	ResponseBody struct {
		// Body is the finalized physical body derived from the whole unary Result.
		Body *expr.AttributeExpr
		// ContentType is an authored encoder selection, empty for negotiation.
		ContentType string
		// ContentTypes lists this body's advertised media independently of SSE.
		ContentTypes []string
		// MediaVariants retains any authored union media contracts.
		MediaVariants []*MediaVariant
		// BodyValue binds the body to the existing service Result occurrence.
		BodyValue *ValueTarget
		// BodyValues retains its per-media preparation contexts.
		BodyValues map[string]*ValueTarget
	}

	ResponseStatus struct {
		// Unary is the separate whole Result representation of typed mixed success.
		Unary *ResponseBody
		// Occurrence identifies the validated response across semantic capture and backends.
		Occurrence *expr.HTTPResponseExpr
		// Carrier is the native result, error, handshake or informational source.
		Carrier *expr.AttributeExpr
		// Handshake marks metadata owned separately from the stream event.
		Handshake bool
		// StatusField is the resolved native status binding, or nil for literal-only responses.
		StatusField *StatusBinding
		// Required retains detached validation-time requirements by native result view.
		Required map[string][]string
		// AuthoredBody is read-only pre-finalization provenance, not another example source.
		AuthoredBody *expr.AttributeExpr
		// MediaVariants retains each typed response representation independently.
		MediaVariants []*MediaVariant
		// BodyValue carries the effective result or error occurrence.
		BodyValue *ValueTarget
		// BodyValues binds each advertised media label to the actual server encoder
		// authority. A mapped header alone does not select that encoder.
		BodyValues map[string]*ValueTarget
		// DocumentValue carries the documentation body authority.
		DocumentValue *ValueTarget
		// DocumentValues records per-media schema authority: actual encoder plans
		// for inherited bodies, declared media contracts for independent bodies.
		DocumentValues map[string]*ValueTarget
		// IndependentDocumentBody marks an authored OpenAPIBody contract.
		IndependentDocumentBody bool
		Error                   *Error
		// Selector is the authored HTTP status owner; JSON-RPC errors leave it unset.
		Selector expr.HTTPStatusSelector
		// Domain is the detached validation-time domain captured by the expression.
		Domain       []int
		Description  string
		ContentType  string
		ContentTypes []string
		Headers      []*Header
		Cookies      []*Cookie
		Body         *expr.AttributeExpr
		DocumentBody *expr.AttributeExpr
		BodyOrigin   string
		// Tag binds response selection to a validated semantic result field.
		Tag          *expr.HTTPResponseTag
		IsError      bool
		EmitExamples bool
		BinaryBody   bool
		Meta         expr.MetaExpr
		Links        []*ResponseLink
		// ErrorSelector records the effective JSON-RPC ErrorName path.
		ErrorSelector []string
		// StatusErrorName is the service field reconstructed from the error
		// name selected by a unique HTTP status, or empty for wire identity.
		StatusErrorName string
	}

	Route struct {
		// Index is the zero-based index of the authored endpoint route.
		Index int
		// OperationIndex is the zero-based index of the emitted route after service base expansion.
		OperationIndex int
		Method         string
		Path           string
		SourcePath     string
		Wildcards      []string
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
		// Codec is the resolved grammar shared by runtime and contract projections.
		Codec httpencoding.Codec
		// Keys identifies this binding's wire namespace.
		Keys httpencoding.KeySet
		// Value shares the selected service member and its actual field codec.
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
		// CredentialBindings retains all effective credential source roles and wire slots.
		CredentialBindings []expr.HTTPCredentialBinding
		Requirements       []*expr.SecurityExpr
		Parameters         []*SecurityParameter
		Disabled           bool
	}

	SecurityParameter struct {
		Name       string
		In         string
		SchemeName string
	}

	Error struct {
		// JSONRPCCode is present only for a named JSON-RPC error mapping.
		JSONRPCCode *int
		Name        string
		Attribute   *expr.AttributeExpr
		Type        expr.DataType
		Remedy      *ErrorRemedy
	}

	ErrorRemedy struct {
		Code        string
		SafeMessage string
		RetryHint   string
	}

	Header struct {
		// Codec is the resolved simple-header grammar.
		Codec httpencoding.Codec
		// Value shares the selected service member and its actual field codec.
		Value            *ValueTarget
		Name             string
		HTTPName         string
		Attribute        *expr.AttributeExpr
		Required         bool
		PrimitivePointer bool
	}

	Cookie struct {
		// Value shares the selected service member and its actual field codec.
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
