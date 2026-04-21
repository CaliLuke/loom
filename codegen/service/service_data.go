package service

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	// ServicesData encapsulates the data computed from the service designs.
	ServicesData struct {
		Root     *expr.RootExpr
		Services map[string]*Data
		// Ctx carries the codegen context used for structured logging and
		// (future) error attribution. A nil Ctx is treated as a silent
		// context; analyze populates it lazily on first use.
		Ctx *codegen.Context
	}

	// Data contains the data used to render the code related to a single
	// service.
	Data struct {
		// Name is the service name.
		Name string
		// Description is the service description.
		Description string
		// APIName is the name of the API the service belongs to.
		APIName string
		// APIVersion is the API version.
		APIVersion string
		// StructName is the service struct name.
		StructName string
		// VarName is the service variable name (first letter in lowercase).
		VarName string
		// PathName is the service name as used in file and import paths.
		PathName string
		// PkgName is the name of the package containing the generated service
		// code.
		PkgName string
		// ViewsPkg is the name of the package containing the projected and viewed
		// result types.
		ViewsPkg string
		// Methods lists the service interface methods.
		Methods []*MethodData
		// Schemes is the list of security schemes required by the service methods.
		Schemes SchemesData
		// ServerInterceptors contains the data needed to render the server-side
		// interceptors code.
		ServerInterceptors []*InterceptorData
		// ClientInterceptors contains the data needed to render the client-side
		// interceptors code.
		ClientInterceptors []*InterceptorData
		// Scope initialized with all the service types.
		Scope *codegen.NameScope
		// ViewScope initialized with all the viewed types.
		ViewScope *codegen.NameScope
		// UserTypeImports lists the import specifications for the user types
		// used by the service.
		UserTypeImports []*codegen.ImportSpec

		// userTypes lists the type definitions that the service depends on.
		userTypes []*UserTypeData
		// errorTypes lists the error type definitions that the service depends on.
		errorTypes []*UserTypeData
		// errorInits list the information required to generate error init
		// functions.
		errorInits []*ErrorInitData
		// projectedTypes lists the types which uses pointers for all fields to
		// define view specific validation logic.
		projectedTypes []*ProjectedTypeData
		// unions lists the sum-type unions defined for the service.
		unions []*UnionTypeData
		// viewedResultTypes lists all the viewed method result types.
		viewedResultTypes []*ViewedResultTypeData
	}

	// MethodData describes a single service method.
	MethodData struct {
		// Name is the method name.
		Name string
		// Description is the method description.
		Description string
		// VarName is the Go method name.
		VarName string
		// Payload contains the payload type metadata.
		MethodPayloadData
		// Result contains the result and viewed-result metadata.
		MethodResultData
		// Security contains the error, auth, and interceptor metadata.
		MethodSecurityData
		// Transport contains the transport and client endpoint-field metadata.
		MethodTransportData
		// Streaming contains the streaming type and stream-interface metadata.
		MethodStreamingData
	}

	// MethodPayloadData contains the request payload type metadata for a method.
	MethodPayloadData struct {
		// Payload is the name of the payload type if any,
		Payload string
		// PayloadLoc defines the file and Go package of the payload type
		// if overridden via Meta.
		PayloadLoc *codegen.Location
		// PayloadDef is the payload type definition if any.
		PayloadDef string
		// PayloadRef is a reference to the payload type if any,
		PayloadRef string
		// PayloadDesc is the payload type description if any.
		PayloadDesc string
		// PayloadEx is an example of a valid payload value.
		PayloadEx any
		// PayloadDefault is the default value of the payload if any.
		PayloadDefault any
	}

	// MethodResultData contains the result type metadata for a method.
	MethodResultData struct {
		// Result is the name of the result type if any.
		Result string
		// ResultLoc defines the file and Go package of the result type
		// if overridden via Meta.
		ResultLoc *codegen.Location
		// ResultDef is the result type definition if any.
		ResultDef string
		// ResultRef is the reference to the result type if any.
		ResultRef string
		// ResultDesc is the result type description if any.
		ResultDesc string
		// ResultEx is an example of a valid result value.
		ResultEx any
		// ViewedResult contains the data required to generate the code handling
		// views if any.
		ViewedResult *ViewedResultTypeData
	}

	// MethodSecurityData contains error, auth, and interceptor metadata.
	MethodSecurityData struct {
		// Errors list the possible errors defined in the design if any.
		Errors []*ErrorInitData
		// ErrorLocs lists the file and Go package of the error type
		// if overridden via Meta indexed by error name.
		ErrorLocs map[string]*codegen.Location
		// Requirements contains the security requirements for the
		// method.
		Requirements RequirementsData
		// Schemes contains the security schemes types used by the
		// method.
		Schemes SchemesData
		// ServerInterceptors list the server interceptors that apply to this
		// method.
		ServerInterceptors []string
		// ClientInterceptors list the client interceptors that apply to this
		// method.
		ClientInterceptors []string
	}

	// MethodTransportData contains transport and client-endpoint metadata.
	MethodTransportData struct {
		// IsJSONRPC indicates if the endpoint is a JSON-RPC endpoint.
		IsJSONRPC bool
		// IsJSONRPCSSE indicates if the JSON-RPC endpoint uses SSE transport.
		IsJSONRPCSSE bool
		// IsJSONRPCWebSocket indicates if the JSON-RPC endpoint uses WebSocket transport.
		IsJSONRPCWebSocket bool
		// SkipRequestBodyEncodeDecode is true if the method payload includes
		// the raw HTTP request body reader.
		SkipRequestBodyEncodeDecode bool
		// SkipResponseBodyEncodeDecode is true if the method result includes
		// the raw HTTP response body reader.
		SkipResponseBodyEncodeDecode bool
		// RequestStruct is the name of the data structure containing the
		// payload and request body reader when SkipRequestBodyEncodeDecode is
		// used.
		RequestStruct string
		// ResponseStruct is the name of the data structure containing the
		// result and response body reader when SkipResponseBodyEncodeDecode is
		// used.
		ResponseStruct string
		// EndpointField is the unique field name used in the generated client
		// struct to store the loom.Endpoint for this method. It is computed with a
		// scope that includes method names to avoid field/method name collisions.
		EndpointField string
		// StreamEndpointField is the unique field name used in the generated client
		// struct to store the "streaming mode" loom.Endpoint for mixed results. The
		// transport endpoint forces server streaming (e.g. sets "Accept:
		// text/event-stream") and returns the client stream interface.
		//
		// It is only set when HasMixedResults is true.
		StreamEndpointField string
	}

	// MethodStreamingData contains streaming type and stream-interface metadata.
	MethodStreamingData struct {
		// StreamingPayload is the name of the streaming payload type if any.
		StreamingPayload string
		// StreamingPayloadDef is the streaming payload type definition if any.
		StreamingPayloadDef string
		// StreamingPayloadRef is a reference to the streaming payload type if any.
		StreamingPayloadRef string
		// StreamingPayloadDesc is the streaming payload type description if any.
		StreamingPayloadDesc string
		// StreamingPayloadEx is an example of a valid streaming payload value.
		StreamingPayloadEx any
		// StreamingResult is the name of the streaming result type if any (when different from Result).
		StreamingResult string
		// StreamingResultDef is the streaming result type definition if any.
		StreamingResultDef string
		// StreamingResultRef is the reference to the streaming result type if any.
		StreamingResultRef string
		// StreamingResultDesc is the streaming result type description if any.
		StreamingResultDesc string
		// StreamingResultEx is an example of a valid streaming result value.
		StreamingResultEx any
		// ServerStream indicates that the service method receives a payload
		// stream or sends a result stream or both.
		ServerStream *StreamData
		// ClientStream indicates that the service method receives a result
		// stream or sends a payload result or both.
		ClientStream *StreamData
		// StreamKind is the kind of the stream (payload or result or
		// bidirectional).
		StreamKind expr.StreamKind
		// HasMixedResults indicates whether the method defines both Result and
		// StreamingResult with different types, enabling content negotiation at
		// the transport layer (e.g. JSON vs SSE over HTTP).
		HasMixedResults bool
	}

	// StreamData is the data used to generate client and server interfaces that
	// a streaming endpoint implements. It is initialized if a method defines a
	// streaming payload or result or both.
	StreamData struct {
		// Interface is the name of the stream interface.
		Interface string
		// VarName is the name of the struct type that implements the stream
		// interface.
		VarName string
		// SendName is the name of the send function.
		SendName string
		// SendDesc is the description for the send function.
		SendDesc string
		// SendWithContextName is the name of the send function with context.
		SendWithContextName string
		// SendWithContextDesc is the description for the send function with context.
		SendWithContextDesc string
		// SendTypeName is the type name sent through the stream.
		SendTypeName string
		// SendTypeRef is the reference to the type sent through the stream.
		SendTypeRef string
		// SendAndCloseName is the name of the send and close function (SSE only).
		SendAndCloseName string
		// SendAndCloseDesc is the description for the send and close function.
		SendAndCloseDesc string
		// SendAndCloseWithContextName is the name of the send and close function with context.
		SendAndCloseWithContextName string
		// SendAndCloseWithContextDesc is the description for the send and close function with context.
		SendAndCloseWithContextDesc string
		// RecvName is the name of the receive function.
		RecvName string
		// RecvDesc is the description for the recv function.
		RecvDesc string
		// RecvWithContextName is the name of the receive function with context.
		RecvWithContextName string
		// RecvWithContextDesc is the description for the recv function with context.
		RecvWithContextDesc string
		// RecvTypeName is the type name received from the stream.
		RecvTypeName string
		// RecvTypeRef is the reference to the type received from the stream.
		RecvTypeRef string
		// MustClose indicates whether the stream should implement the Close()
		// function.
		MustClose bool
		// EndpointStruct is the name of the endpoint struct that holds a payload
		// reference (if any) and the endpoint server stream.
		EndpointStruct string
		// Kind is the kind of the stream (payload, result or bidirectional).
		Kind expr.StreamKind
	}

	// ErrorInitData describes an error returned by a service method of type
	// ErrorResult.
	ErrorInitData struct {
		// Name is the name of the init function.
		Name string
		// Description is the error description.
		Description string
		// ErrName is the name of the error.
		ErrName string
		// TypeName is the error struct type name.
		TypeName string
		// TypeRef is the reference to the error type.
		TypeRef string
		// Temporary indicates whether the error is temporary.
		Temporary bool
		// Timeout indicates whether the error is due to timeouts.
		Timeout bool
		// Fault indicates whether the error is server-side fault.
		Fault bool
		// RemedyCode is the stable remediation code for the error if declared.
		RemedyCode string
		// SafeMessage is the safe, user-facing message for the error if declared.
		SafeMessage string
		// RetryHint is the retry or correction guidance for the error if declared.
		RetryHint string
	}

	// InterceptorData contains the data required to render the service-level
	// interceptor code. interceptors.go.tpl
	InterceptorData struct {
		// Name is the name of the interceptor used in the generated code.
		Name string
		// DesignName is the name of the interceptor as defined in the design.
		DesignName string
		// Description is the description of the interceptor from the design.
		Description string
		// Methods
		Methods []*MethodInterceptorData
		// ReadPayload contains payload attributes that the interceptor can
		// read.
		ReadPayload []*AttributeData
		// WritePayload contains payload attributes that the interceptor can
		// write.
		WritePayload []*AttributeData
		// ReadResult contains result attributes that the interceptor can read.
		ReadResult []*AttributeData
		// WriteResult contains result attributes that the interceptor can
		// write.
		WriteResult []*AttributeData
		// ReadStreamingPayload contains streaming payload attributes that the interceptor can read.
		ReadStreamingPayload []*AttributeData
		// WriteStreamingPayload contains streaming payload attributes that the interceptor can write.
		WriteStreamingPayload []*AttributeData
		// ReadStreamingResult contains streaming result attributes that the interceptor can read.
		ReadStreamingResult []*AttributeData
		// WriteStreamingResult contains streaming result attributes that the interceptor can write.
		WriteStreamingResult []*AttributeData
		// HasPayloadAccess indicates that the interceptor info object has a
		// payload access interface.
		HasPayloadAccess bool
		// HasResultAccess indicates that the interceptor info object has a
		// result access interface.
		HasResultAccess bool
		// HasStreamingPayloadAccess indicates that the interceptor info object has a
		// streaming payload access interface.
		HasStreamingPayloadAccess bool
		// HasStreamingResultAccess indicates that the interceptor info object has a
		// streaming result access interface.
		HasStreamingResultAccess bool
	}

	// MethodInterceptorData contains the data required to render the
	// method-level interceptor code.
	MethodInterceptorData struct {
		// MethodName is the name of the method.
		MethodName string
		// PayloadAccess is the name of the payload access struct.
		PayloadAccess string
		// ResultAccess is the name of the result access struct.
		ResultAccess string
		// StreamingPayloadAccess is the name of the streaming payload access struct.
		StreamingPayloadAccess string
		// StreamingResultAccess is the name of the streaming result access struct.
		StreamingResultAccess string
		// PayloadRef is the reference to the method payload type.
		PayloadRef string
		// ResultRef is the reference to the method result type.
		ResultRef string
		// StreamingPayloadRef is the reference to the streaming payload type.
		StreamingPayloadRef string
		// StreamingResultRef is the reference to the streaming result type.
		StreamingResultRef string
		// ServerStream is the stream data if the endpoint defines a server stream.
		ServerStream *StreamInterceptorData
		// ClientStream is the stream data if the endpoint defines a client stream.
		ClientStream *StreamInterceptorData
	}

	// StreamInterceptorData is the stream data for an interceptor.
	StreamInterceptorData struct {
		// Interface is the name of the stream interface.
		Interface string
		// SendName is the name of the send function.
		SendName string
		// SendWithContextName is the name of the send function with context.
		SendWithContextName string
		// SendTypeRef is the reference to the type sent through the stream.
		SendTypeRef string
		// RecvName is the name of the recv function.
		RecvName string
		// RecvWithContextName is the name of the recv function with context.
		RecvWithContextName string
		// RecvTypeRef is the reference to the type received from the stream.
		RecvTypeRef string
		// MustClose indicates whether the stream should implement the Close()
		// function.
		MustClose bool
		// EndpointStruct is the name of the endpoint struct that holds a payload
		// reference (if any) and the endpoint server stream.
		EndpointStruct string
	}

	// AttributeData describes a single attribute.
	AttributeData struct {
		// Name is the name of the attribute.
		Name string
		// TypeRef is the reference to the attribute type.
		TypeRef string
		// Pointer is true if the attribute is a pointer.
		Pointer bool
	}

	// RequirementsData is the list of security requirements.
	RequirementsData []*RequirementData

	// SchemesData is the list of security schemes.
	SchemesData []*SchemeData

	// RequirementData lists the schemes and scopes defined by a single
	// security requirement.
	RequirementData struct {
		// Schemes list the requirement schemes.
		Schemes []*SchemeData
		// Scopes list the required scopes.
		Scopes []string
	}

	// UserTypeData contains the data describing a user-defined type.
	UserTypeData struct {
		// Name is the type name.
		Name string
		// VarName is the corresponding Go type name.
		VarName string
		// Description is the type human description.
		Description string
		// Def is the type definition Go code.
		Def string
		// Ref is the reference to the type.
		Ref string
		// Loc defines the file and Go package of the type if overridden
		// via Meta.
		Loc *codegen.Location
		// Type is the underlying type.
		Type expr.UserType
		// RemedyCode is the stable remediation code for this error type if declared.
		RemedyCode string
		// SafeMessage is the safe, user-facing message for this error type if declared.
		SafeMessage string
		// RetryHint is retry or correction guidance for this error type if declared.
		RetryHint string
	}

	// UnionTypeData describes a generated sum-type union for a service.
	UnionTypeData struct {
		// Name is the Go type name of the union struct.
		Name string
		// KindName is the Go type name of the discriminator kind.
		KindName string
		// Fields describes each union branch.
		Fields []*UnionFieldData
		// Loc defines the file and Go package of the union type if overridden via
		// Meta. When nil the type is generated in the default service file.
		Loc *codegen.Location
		// TypeKey is the discriminator field name for JSON marshaling (defaults to "type").
		TypeKey string
		// ValueKey is the value field name for JSON marshaling (defaults to "value").
		ValueKey string
		// HasScalarFormBranch is true when at least one branch keeps canonical
		// type/value form encoding.
		HasScalarFormBranch bool
	}

	// UnionFieldData describes a single branch of a union.
	UnionFieldData struct {
		// Name is the branch name as defined in the DSL.
		Name string
		// KindConst is the Go identifier for the kind constant of this branch.
		KindConst string
		// FieldName is the struct field name in the union.
		FieldName string
		// FieldType is the Go type used in the union struct field and public API.
		FieldType string
		// FlatFormObject is true when the branch value is object-shaped and form
		// encoding should flatten its fields under the current prefix.
		FlatFormObject bool
		// FlatFormObjectAllowsEmpty is true when the flattened object branch may be
		// selected with only the discriminator because it has no required fields.
		FlatFormObjectAllowsEmpty bool
		// EmptyValueExpr is the Go expression that initializes an empty branch
		// value when FlatFormObjectAllowsEmpty is true.
		EmptyValueExpr string
		// EmitPrimitiveAlias is true when the branch uses a generated primitive alias
		// that must be declared in the same file as the union type.
		EmitPrimitiveAlias bool
		// PrimitiveAliasType is the underlying Go type used by the generated branch
		// alias (for example "string" or "float64").
		PrimitiveAliasType string
		// TypeTag is the JSON "type" discriminator value for this branch.
		TypeTag string
	}

	// SchemeData describes a single security scheme.
	SchemeData struct {
		// Kind is the type of scheme, one of "Basic", "APIKey", "JWT"
		// or "OAuth2".
		Type string
		// SchemeName is the name of the scheme.
		SchemeName string
		// Name refers to a header or parameter name, based on In's
		// value.
		Name string
		// UsernameField is the name of the payload field that should be
		// initialized with the basic auth username if any.
		UsernameField string
		// UsernamePointer is true if the username field is a pointer.
		UsernamePointer bool
		// UsernameAttr is the name of the attribute that contains the
		// username.
		UsernameAttr string
		// UsernameRequired specifies whether the attribute that
		// contains the username is required.
		UsernameRequired bool
		// PasswordField is the name of the payload field that should be
		// initialized with the basic auth password if any.
		PasswordField string
		// PasswordPointer is true if the password field is a pointer.
		PasswordPointer bool
		// PasswordAttr is the name of the attribute that contains the
		// password.
		PasswordAttr string
		// PasswordRequired specifies whether the attribute that
		// contains the password is required.
		PasswordRequired bool
		// CredField contains the name of the payload field that should
		// be initialized with the API key, the JWT token or the OAuth2
		// access token.
		CredField string
		// CredPointer is true if the credential field is a pointer.
		CredPointer bool
		// CredRequired specifies if the key is a required attribute.
		CredRequired bool
		// KeyAttr is the name of the attribute that contains
		// the security tag (for APIKey, OAuth2, and JWT schemes).
		KeyAttr string
		// Scopes lists the scopes that apply to the scheme.
		Scopes []string
		// Flows describes the OAuth2 flows.
		Flows []*expr.FlowExpr
		// In indicates the request element that holds the credential.
		In string
		// TransportOwned is true when the credential is supplied by transport
		// state rather than a payload field.
		TransportOwned bool
	}
)

// NewServicesData creates a new ServicesData instance for the given root.
// The returned ServicesData has a codegen context whose logger honors the
// DEBUG_LOOM environment variable (silent when unset). Callers that want to
// force silence regardless of environment can assign [codegen.NewSilentContext]
// to the Ctx field.
func NewServicesData(root *expr.RootExpr) *ServicesData {
	return &ServicesData{
		Services: make(map[string]*Data),
		Root:     root,
		Ctx:      codegen.NewContext(),
	}
}

// Get retrieves the data for the service with the given name computing it if
// needed. It returns nil if there is no service with the given name.
func (d *ServicesData) Get(name string) *Data {
	if data, ok := d.Services[name]; ok {
		return data
	}
	service := d.Root.Service(name)
	if service == nil {
		return nil
	}
	d.Services[name] = d.analyze(service)
	return d.Services[name]
}

// Method returns the service method data for the method with the given name,
// nil if there isn't one.
func (d *Data) Method(name string) *MethodData {
	for _, m := range d.Methods {
		if m.Name == name {
			return m
		}
	}
	return nil
}

// SetViewedResult records the viewed-result wrapper metadata for the method.
func (m *MethodData) SetViewedResult(viewed *ViewedResultTypeData) {
	m.MethodResultData.ViewedResult = viewed
}

// AssignEndpointFields computes the generated client endpoint field names for
// the method.
func (m *MethodData) AssignEndpointFields(scope *codegen.NameScope) {
	m.MethodTransportData.EndpointField = scope.Unique(m.VarName+"Endpoint", "")
	if m.HasMixedResults {
		m.MethodTransportData.StreamEndpointField = scope.Unique(m.VarName+"StreamEndpoint", "")
	}
}

// AppendInterceptorName records an interceptor name on the server or client
// side of the method.
func (m *MethodData) AppendInterceptorName(name string, server bool) {
	if server {
		m.MethodSecurityData.ServerInterceptors = append(m.MethodSecurityData.ServerInterceptors, name)
		return
	}
	m.MethodSecurityData.ClientInterceptors = append(m.MethodSecurityData.ClientInterceptors, name)
}

// Scheme returns the scheme data with the given scheme name.
func (r RequirementsData) Scheme(name string) *SchemeData {
	for _, req := range r {
		for _, s := range req.Schemes {
			if s.SchemeName == name {
				return s
			}
		}
	}
	return nil
}

// Dup creates a copy of the scheme data.
func (s *SchemeData) Dup() *SchemeData {
	return &SchemeData{
		Type:             s.Type,
		SchemeName:       s.SchemeName,
		Name:             s.Name,
		UsernameField:    s.UsernameField,
		UsernamePointer:  s.UsernamePointer,
		UsernameAttr:     s.UsernameAttr,
		UsernameRequired: s.UsernameRequired,
		PasswordField:    s.PasswordField,
		PasswordPointer:  s.PasswordPointer,
		PasswordAttr:     s.PasswordAttr,
		PasswordRequired: s.PasswordRequired,
		CredField:        s.CredField,
		CredPointer:      s.CredPointer,
		CredRequired:     s.CredRequired,
		KeyAttr:          s.KeyAttr,
		Scopes:           s.Scopes,
		Flows:            s.Flows,
		In:               s.In,
		TransportOwned:   s.TransportOwned,
	}
}

// Append appends a scheme data to schemes only if it doesn't exist.
func (s SchemesData) Append(d *SchemeData) SchemesData {
	if d == nil {
		return s
	}
	found := false
	for _, se := range s {
		if se.SchemeName == d.SchemeName {
			found = true
			break
		}
	}
	if found {
		return s
	}
	return append(s, d)
}

// DedupeByType returns a new SchemesData slice that is deduplicated by scheme
// type.
func (s SchemesData) DedupeByType() SchemesData {
	seen := make(map[string]struct{})
	uniqueSchemes := SchemesData{}
	for _, s := range s {
		if _, ok := seen[s.Type]; !ok {
			seen[s.Type] = struct{}{}
			uniqueSchemes = append(uniqueSchemes, s)
		}
	}

	return uniqueSchemes
}
