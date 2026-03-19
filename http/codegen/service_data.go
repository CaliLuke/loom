package codegen

import (
	"bytes"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/codegen/service"
	"goa.design/goa/v3/expr"
)

var (
	// pathInitTmpl is the template used to render path constructors code.
	pathInitTmpl = template.Must(
		template.New("path-init").
			Funcs(template.FuncMap{"goify": codegen.Goify}).
			Parse(httpTemplates.Read(pathInitT, querySliceConversionP)),
	)
)

type (
	// ServicesData encapsulates the data computed from the design.
	ServicesData struct {
		*service.ServicesData
		Expressions *expr.HTTPExpr
		HTTPData    map[string]*ServiceData
	}

	// ServiceData contains the data used to render the code related to a
	// single service.
	ServiceData struct {
		// Service contains the related service data.
		Service *service.Data
		// Endpoints describes the endpoint data for this service.
		Endpoints []*EndpointData
		// FileServers lists the file servers for this service.
		FileServers []*FileServerData
		// ServerStruct is the name of the HTTP server struct.
		ServerStruct string
		// MountPointStruct is the name of the mount point struct.
		MountPointStruct string
		// ServerInit is the name of the constructor of the server
		// struct.
		ServerInit string
		// MountServer is the name of the mount function.
		MountServer string
		// ServerService is the name of service function.
		ServerService string
		// ClientStruct is the name of the HTTP client struct.
		ClientStruct string
		// ServerBodyAttributeTypes is the list of user types used to
		// define the request, response and error response type
		// attributes in the server code.
		ServerBodyAttributeTypes []*TypeData
		// ClientBodyAttributeTypes is the list of user types used to
		// define the request, response and error response type
		// attributes in the client code.
		ClientBodyAttributeTypes []*TypeData
		// ServerTypeNames records the user type names used to define
		// the endpoint request and response bodies for server code.
		// The type name is used as the key and a bool as the value
		// which if true indicates that the type has been generated
		// in the server package.
		ServerTypeNames map[string]bool
		// ClientTypeNames records the user type names used to define
		// the endpoint request and response bodies for client code.
		// The type name is used as the key and a bool as the value
		// which if true indicates that the type has been generated
		// in the client package.
		ClientTypeNames map[string]bool
		// ServerTransformHelpers is the list of transform functions
		// required by the various server side constructors.
		ServerTransformHelpers []*codegen.TransformFunctionData
		// ClientTransformHelpers is the list of transform functions
		// required by the various client side constructors.
		ClientTransformHelpers []*codegen.TransformFunctionData
		// UnionTypes lists the sum-type unions referenced by the HTTP request and
		// response body types.
		UnionTypes []*service.UnionTypeData
		// Scope initialized with all the server and client types.
		Scope *codegen.NameScope
	}

	// EndpointData contains the data used to render the code related to a
	// single service HTTP endpoint.
	EndpointData struct {
		// Method contains the related service method data.
		Method *service.MethodData
		// ServiceName is the name of the service exposing the endpoint.
		ServiceName string
		// ServiceVarName is the goified service name (first letter
		// lowercase).
		ServiceVarName string
		// ServicePkgName is the name of the service package.
		ServicePkgName string
		// Payload describes the method HTTP payload.
		Payload *PayloadData
		// Result describes the method HTTP result.
		Result *ResultData
		// Errors describes the method HTTP errors.
		Errors []*ErrorGroupData
		// Routes describes the possible routes for this endpoint.
		Routes []*RouteData
		// BasicScheme is the basic auth security scheme if any.
		BasicScheme *service.SchemeData
		// HeaderSchemes lists all the security requirement schemes that
		// apply to the method and are encoded in the request header.
		HeaderSchemes service.SchemesData
		// BodySchemes lists all the security requirement schemes that
		// apply to the method and are encoded in the request body.
		BodySchemes service.SchemesData
		// QuerySchemes lists all the security requirement schemes that
		// apply to the method and are encoded in the request query
		// string.
		QuerySchemes service.SchemesData
		// Requirements contains the security requirements for the
		// method.
		Requirements service.RequirementsData

		// server

		// MountHandler is the name of the mount handler function.
		MountHandler string
		// HandlerInit is the name of the constructor function for the
		// http handler function.
		HandlerInit string
		// RequestDecoder is the name of the request decoder function.
		RequestDecoder string
		// ResponseEncoder is the name of the response encoder function.
		ResponseEncoder string
		// ErrorEncoder is the name of the error encoder function.
		ErrorEncoder string
		// MultipartRequestDecoder indicates the request decoder for
		// multipart content type.
		MultipartRequestDecoder *MultipartData
		// ServerWebSocket holds the data to render the server struct which
		// implements the server stream interface.
		ServerWebSocket *WebSocketData
		// SSE holds the data to render the server struct which implements the
		// server stream interface for SSE.
		SSE *SSEData
		// Redirect defines a redirect for the endpoint.
		Redirect *RedirectData
		// HasMixedResults indicates if the method has both Result and StreamingResult
		// defined with different types, enabling content negotiation.
		HasMixedResults bool

		// client

		// ClientStruct is the name of the HTTP client struct.
		ClientStruct string
		// EndpointInit is the name of the constructor function for the
		// client endpoint.
		EndpointInit string
		// RequestInit is the request builder function.
		RequestInit *InitData
		// RequestEncoder is the name of the request encoder function.
		RequestEncoder string
		// ResponseDecoder is the name of the response decoder function.
		ResponseDecoder string
		// MultipartRequestEncoder indicates the request encoder for
		// multipart content type.
		MultipartRequestEncoder *MultipartData
		// ClientWebSocket holds the data to render the client struct which
		// implements the client stream interface.
		ClientWebSocket *WebSocketData
		// BuildStreamPayload is the name of the function used to create the
		// payload for endpoints that use SkipRequestBodyEncodeDecode.
		BuildStreamPayload string
	}

	// FileServerData lists the data needed to generate file servers.
	FileServerData struct {
		// MountHandler is the name of the mount handler function.
		MountHandler string
		// RequestPaths is the set of HTTP paths to the server.
		RequestPaths []string
		// Root is the root server file path.
		FilePath string
		// Dir is true if the file server servers files under a
		// directory, false if it serves a single file.
		IsDir bool
		// PathParam is the name of the parameter used to capture the
		// path for file servers that serve files under a directory.
		PathParam string
		// Redirect defines a redirect for the endpoint.
		Redirect *RedirectData
		// VarName is the name of the variable that holds the file server.
		VarName string
		// ArgName is the name of the argument used to initialize the
		// file server.
		ArgName string
	}

	// RedirectData lists the data needed to generate a redirect.
	RedirectData struct {
		// URL is the URL that is being redirected to.
		URL string
		// StatusCode is the HTTP status code.
		StatusCode string
	}

	// PayloadData contains the payload information required to generate the
	// transport decode (server) and encode (client) code.
	PayloadData struct {
		// Name is the name of the payload type.
		Name string
		// Ref is the fully qualified reference to the payload type.
		Ref string
		// Request contains the data for the corresponding HTTP request.
		Request *RequestData
		// DecoderReturnValue is a reference to the decoder return value
		// if there is no payload constructor (i.e. if Init is nil).
		DecoderReturnValue string
		// IDAttribute is the name of the attribute where the ID of the
		// JSON-RPC request is stored.
		IDAttribute string
		// IDAttributeRequired is true if the ID attribute is required.
		IDAttributeRequired bool
	}

	// ResultData contains the result information required to generate the
	// transport decode (client) and encode (server) code.
	ResultData struct {
		// Name is the name of the result type.
		Name string
		// Ref is the reference to the result type.
		Ref string
		// IsStruct is true if the result type is a user type defining
		// an object.
		IsStruct bool
		// Inits contains the data required to render the result
		// constructors if any.
		Inits []*InitData
		// Responses contains the data for the corresponding HTTP
		// responses.
		Responses []*ResponseData
		// IDAttribute is the name of the attribute where the ID of the
		// JSON-RPC request is stored.
		IDAttribute string
		// IDAttributeRequired is true if the ID attribute is required.
		IDAttributeRequired bool
		// View is the view used to render the result.
		View string
		// MustInit indicates if a variable holding the result type must be
		// initialized. It is used by server response encoder to initialize
		// the result variable only if there are multiple responses, or the
		// response has a body, a header or a cookie.
		MustInit bool
	}

	// ErrorGroupData contains the error information required to generate
	// the transport decode (client) and encode (server) code for all errors
	// with responses using a given HTTP status code.
	ErrorGroupData struct {
		// StatusCode is the response HTTP status code.
		StatusCode string
		// Errors contains the information for each error.
		Errors []*ErrorData
	}

	// ErrorData contains the error information required to generate the
	// transport decode (client) and encode (server) code.
	ErrorData struct {
		// Name is the error name.
		Name string
		// Ref is a reference to the error type.
		Ref string
		// Response is the error response data.
		Response *ResponseData
	}

	// RequestData describes a request.
	RequestData struct {
		// PathParams describes the information about params that are
		// present in the request path.
		PathParams []*ParamData
		// QueryParams describes the information about the params that
		// are present in the request query string.
		QueryParams []*ParamData
		// Headers contains the HTTP request headers used to build the
		// method payload.
		Headers []*HeaderData
		// Cookies contains the HTTP request cookies used to build the
		// method payload.
		Cookies []*CookieData
		// ServerBody describes the request body type used by server
		// code. The type is generated using pointers for all fields so
		// that it can be validated.
		ServerBody *TypeData
		// ClientBody describes the request body type used by client
		// code. The type does NOT use pointers for every fields since
		// no validation is required.
		ClientBody *TypeData
		// PayloadInit contains the data required to render the
		// payload constructor used by server code if any.
		PayloadInit *InitData
		// PayloadType is the type of the payload.
		PayloadType expr.DataType
		// PayloadAttr sets the request body from the specified payload type
		// attribute. This field is set when the design uses Body("name") syntax
		// to set the request body and the payload type is an object.
		PayloadAttr string
		// MustHaveBody is true if the request body cannot be empty.
		MustHaveBody bool
		// MustValidate is true if the request body or at least one
		// parameter or header requires validation.
		MustValidate bool
		// Multipart if true indicates the request is a multipart
		// request.
		Multipart bool
		// MultipartGenerated is true when multipart request decoding is fully
		// generated by the framework for the request body shape.
		MultipartGenerated bool
		// MultipartFileFields describes multipart file parts that need explicit
		// handling during generated request decoding.
		MultipartFileFields []*MultipartFileFieldData
		// FormEncoded if true indicates the request uses
		// application/x-www-form-urlencoded.
		FormEncoded bool
	}

	// MultipartFileFieldData describes a multipart file field handled by the
	// generated request decoder.
	MultipartFileFieldData struct {
		// Name is the payload/body attribute name.
		Name string
		// HTTPName is the multipart form field name.
		HTTPName string
		// Required indicates whether the file field is required.
		Required bool
		// PopulateFilename indicates whether the generated decoder should
		// auto-populate the sibling "filename" field from the uploaded part.
		PopulateFilename bool
		// PopulateContentType indicates whether the generated decoder should
		// auto-populate the sibling "content_type" field from the uploaded part.
		PopulateContentType bool
	}

	// ResponseData describes a response.
	ResponseData struct {
		// StatusCode is the return code of the response.
		StatusCode string
		// Code is the return code of the response.
		Code int
		// Description is the response description.
		Description string
		// Headers provides information about the HTTP response headers.
		Headers []*HeaderData
		// Cookies provides information about the HTTP response cookies.
		Cookies []*CookieData
		// ContentType contains the value of the response
		// "Content-Type" header.
		ContentType string
		// ErrorHeader contains the value of the response "goa-error"
		// header if any.
		ErrorHeader string
		// ServerBody is the type of the response body used by server
		// code, nil if body should be empty. The type does NOT use
		// pointers for all fields. If the method result is a result
		// type and the response data describes a success response, then
		// this field contains a type for every view in the result type.
		// The type name is suffixed with the name of the view (except
		// for "default" view where no suffix is added). A constructor
		// is also generated server side for each view to transform the
		// result type to the corresponding response body type. If
		// method result is not a result type or if the response
		// describes an error response, then this field contains at most
		// one item.
		ServerBody []*TypeData
		// ClientBody is the type of the response body used by client
		// code, nil if body should be empty. The type uses pointers for
		// all fields so they can be validated.
		ClientBody *TypeData
		// Init contains the data required to render the result or error
		// constructor if any.
		ResultInit *InitData
		// TagName is the name of the attribute used to test whether the
		// response is the one to use.
		TagName string
		// TagValue is the value the result attribute named by TagName
		// must have for this response to be used.
		TagValue string
		// TagPointer is true if the tag attribute is a pointer.
		TagPointer bool
		// MustValidate is true if at least one header requires validation.
		MustValidate bool
		// ResultAttr sets the response body from the specified result
		// type attribute. This field is set when the design uses
		// Body("name") syntax to set the response body and the result
		// type is an object.
		ResultAttr string
		// ViewedResult indicates whether the response body type is a
		// result type.
		ViewedResult *service.ViewedResultTypeData
	}

	// InitData contains the data required to render a constructor.
	InitData struct {
		// Name is the constructor function name.
		Name string
		// Description is the function description.
		Description string
		// ServerArgs is the list of constructor arguments for server
		// side code.
		ServerArgs []*InitArgData
		// ClientArgs is the list of constructor arguments for client
		// side code.
		ClientArgs []*InitArgData
		// CLIArgs is the list of arguments that should be initialized
		// from CLI flags. This is used for implicit attributes which
		// as the time of writing is only used for the basic auth
		// username and password.
		CLIArgs []*InitArgData
		// ServerCode is the code that builds the payload from the
		// request on the server when it contains user types.
		ServerCode string
		// ClientCode is the code that builds the payload or result type
		// from the request or response state on the client when it
		// contains user types.
		ClientCode string
		// ReturnTypePkg is the package where the return type is present.
		ReturnTypePkg string
		// ReturnTypeName is the qualified (including the package name)
		// name of the payload, result or error type.
		ReturnTypeName string
		// ReturnTypeRef is the qualified (including the package name)
		// reference to the payload, result or error type.
		ReturnTypeRef string
		// ReturnTypeAttribute is the name of the attribute initialized by this
		// constructor when it only initializes one attribute (i.e. body was
		// defined with Body("name") syntax).
		ReturnTypeAttribute string
		// ReturnIsStruct is true if the payload, result or error type is a struct.
		ReturnIsStruct bool
		// ReturnIsPrimitivePointer indicates whether the payload, result or error
		// type is a primitive pointer.
		ReturnIsPrimitivePointer bool
	}

	// AttributeData contains the information needed to generate the code
	// related to a specific payload or result attribute.
	AttributeData struct {
		// Name is the name of the attribute.
		Name string
		// VarName is the name of the variable that holds the attribute value.
		VarName string
		// Pointer is true if the attribute value is a pointer.
		Pointer bool
		// Required is true if the attribute is required in the parent attribute.
		Required bool
		// Type is the attribute type.
		Type expr.DataType
		// TypeName is the generated attribute type name.
		TypeName string
		// TypeRef is the generated attribute type reference.
		TypeRef string
		// Description is the attribute description as defined in the design.
		Description string
		// FieldName is the name of the data structure field that should
		// be initialized with the value if any.
		FieldName string
		// FieldType is the type of the data structure field that should be
		// initialized with the attribute value or read into the attribute value.
		FieldType expr.DataType
		// FieldPointer if true indicates that the data structure field is a
		// pointer.
		FieldPointer bool
		// DefaultValue is the default value of the attribute if any.
		DefaultValue any
		// Validate contains the validation code for the attribute value if any.
		Validate string
		// Example is an example attribute value
		Example any
		// IsAliased is true if the field type is a user-defined type (alias).
		IsAliased bool
		// ServiceTypeRef is the service-aware type reference for cross-service resolution.
		ServiceTypeRef string
	}

	// InitArgData represents a single constructor argument.
	InitArgData struct {
		*AttributeData
		// Reference to the argument, e.g. "&body".
		Ref string
	}

	// RouteData describes a route.
	RouteData struct {
		// Verb is the HTTP method.
		Verb string
		// Path is the fullpath including wildcards.
		Path string
		// PathInit contains the information needed to render and call
		// the path constructor for the route.
		PathInit *InitData
	}

	// Element defines the common fields needed to generate HTTP request and
	// response elements including headers, parameters and cookies.
	Element struct {
		*AttributeData
		// HTTPName is the name of the HTTP element (header name, query string name
		// or cookie name)
		HTTPName string
		// AttributeName is the name of the corresponding attribute.
		AttributeName string
		// StringSlice is true if the attribute type is array of strings.
		StringSlice bool
		// Slice is true if the attribute type is an array.
		Slice bool
	}

	// ParamData describes a HTTP request parameter (query string or path
	// parameter).
	ParamData struct {
		*Element
		// MapStringSlice is true if the param type is a map of string
		// slice.
		MapStringSlice bool
		// Map is true if the param type is a map.
		Map bool
		// MapQueryParams indicates that the query params must be mapped
		// to the entire payload (empty string) or a payload attribute
		// (attribute name).
		MapQueryParams *string
	}

	// HeaderData describes a HTTP request or response header.
	HeaderData struct {
		*Element
		// CanonicalName is the canonical header key.
		CanonicalName string
	}

	// CookieData describes a HTTP request or response cookie.
	CookieData struct {
		*Element
		// MaxAge is the cookie "max-age" attribute.
		MaxAge string
		// Path is the cookie "path" attribute.
		Path string
		// Domain is the cookie "domain" attribute.
		Domain string
		// Secure sets the cookie "secure" attribute to "Secure" if true.
		Secure bool
		// HTTPOnly sets the cookie "http-only" attribute to "HttpOnly" if true.
		HTTPOnly bool
		// SameSite sets the cookie "same-site" attribute to the given value.
		SameSite string
	}

	// TypeData contains the data needed to render a type definition.
	TypeData struct {
		// Name is the type name.
		Name string
		// VarName is the Go type name.
		VarName string
		// Description is the type human description.
		Description string
		// Init contains the data needed to render and call the type
		// constructor if any.
		Init *InitData
		// Def is the type definition Go code.
		Def string
		// Ref is the reference to the type.
		Ref string
		// ValidateDef contains the validation code.
		ValidateDef string
		// ValidateRef contains the call to the validation code.
		ValidateRef string
		// Example is an example value for the type.
		Example any
		// View is the view used to render the (result) type if any.
		View string
		// FlatFormUnionField is the field name of a synthetic single-field request
		// body wrapper that should delegate form encoding and decoding to the
		// wrapped union directly.
		FlatFormUnionField string
	}

	// MultipartData contains the data needed to render multipart
	// encoder/decoder.
	MultipartData struct {
		// FuncName is the name used to generate function type.
		FuncName string
		// InitName is the name of the constructor.
		InitName string
		// VarName is the name of the variable referring to the function.
		VarName string
		// ServiceName is the name of the service.
		ServiceName string
		// MethodName is the name of the method.
		MethodName string
		// Payload is the payload data required to generate
		// encoder/decoder.
		Payload *PayloadData
	}
)

// NewServicesData creates a new ServicesData instance for the given service data.
func NewServicesData(services *service.ServicesData, expressions *expr.HTTPExpr) *ServicesData {
	return &ServicesData{
		ServicesData: services,
		Expressions:  expressions,
		HTTPData:     make(map[string]*ServiceData),
	}
}

// Get retrieves the transport data for the service with the given name
// computing it if needed. It returns nil if there is no service with the given
// name.
func (sds *ServicesData) Get(name string) *ServiceData {
	if data, ok := sds.HTTPData[name]; ok {
		return data
	}
	svc := sds.Expressions.Service(name)
	if svc == nil {
		return nil
	}
	sds.HTTPData[name] = sds.analyze(svc)
	return sds.HTTPData[name]
}

// Endpoint returns the service method transport data for the endpoint with the
// given name, nil if there isn't one.
func (svc *ServiceData) Endpoint(name string) *EndpointData {
	for _, e := range svc.Endpoints {
		if e.Method.Name == name {
			return e
		}
	}
	return nil
}

// analyze creates the data necessary to render the code of the given service.
// It records the user types needed by the service definition in userTypes.
func (sds *ServicesData) analyze(httpSvc *expr.HTTPServiceExpr) *ServiceData {
	svc := sds.ServicesData.Get(httpSvc.ServiceExpr.Name)
	scope := codegen.NewNameScope()
	scope.Unique("c") // 'c' is reserved as the client's receiver name.
	scope.Unique("v") // 'v' is reserved as the request builder payload argument name.
	// Reserve 'websocket' to avoid collision with gorilla/websocket
	scope.Unique("websocket")
	// Reserve the service package name to avoid collision with parameter names in generated code
	scope.Unique(svc.PkgName)
	sd := &ServiceData{
		Service:          svc,
		ServerStruct:     "Server",
		MountPointStruct: "MountPoint",
		ServerInit:       "New",
		MountServer:      "Mount",
		ServerService:    "Service",
		ClientStruct:     "Client",
		ServerTypeNames:  make(map[string]bool),
		ClientTypeNames:  make(map[string]bool),
		Scope:            scope,
	}
	sd.FileServers = sds.buildFileServersData(httpSvc, scope)
	for _, httpEndpoint := range httpSvc.HTTPEndpoints {
		sd.Endpoints = append(sd.Endpoints, sds.buildEndpointData(httpEndpoint, svc, sd, scope))
	}
	for _, httpEndpoint := range httpSvc.HTTPEndpoints {
		sds.collectEndpointBodyAttributeTypes(httpEndpoint, sd)
	}
	sd.UnionTypes = sds.collectEndpointUnionTypes(httpSvc, sd.Scope)

	return sd
}

func (sds *ServicesData) buildFileServersData(httpSvc *expr.HTTPServiceExpr, scope *codegen.NameScope) []*FileServerData {
	fileServers := make([]*FileServerData, 0, len(httpSvc.FileServers))
	for _, server := range httpSvc.FileServers {
		paths := make([]string, len(server.RequestPaths))
		for i, path := range server.RequestPaths {
			idx := strings.LastIndex(path, "/{")
			switch {
			case idx == 0:
				paths[i] = "/"
			case idx > 0:
				paths[i] = path[:idx]
			default:
				paths[i] = path
			}
		}
		var pathParam string
		if server.IsDir() {
			pathParam = expr.ExtractHTTPWildcards(server.RequestPaths[0])[0]
		}
		var redirect *RedirectData
		if server.Redirect != nil {
			redirect = &RedirectData{
				URL:        server.Redirect.URL,
				StatusCode: statusCodeToHTTPConst(server.Redirect.StatusCode),
			}
		}
		fileServers = append(fileServers, &FileServerData{
			MountHandler: scope.Unique(fmt.Sprintf("Mount%s", codegen.Goify(server.FilePath, true))),
			RequestPaths: paths,
			FilePath:     server.FilePath,
			IsDir:        server.IsDir(),
			PathParam:    pathParam,
			Redirect:     redirect,
			VarName:      scope.Unique(codegen.Goify(server.FilePath, true)),
			ArgName:      scope.Unique(fmt.Sprintf("fileSystem%s", codegen.Goify(server.FilePath, true))),
		})
	}
	return fileServers
}

func (sds *ServicesData) buildEndpointData(httpEndpoint *expr.HTTPEndpointExpr, svc *service.Data, sd *ServiceData, scope *codegen.NameScope) *EndpointData {
	method := svc.Method(httpEndpoint.MethodExpr.Name)
	routes := sds.buildEndpointRoutes(httpEndpoint, method, svc, sd)
	payload := sds.buildPayloadData(httpEndpoint, sd)
	reqs, hsch, bosch, qsch, basch := sds.buildRequirementSchemes(httpEndpoint)
	requestInit := sds.buildClientRequestInit(httpEndpoint, method, svc, routes, sd)

	endpoint := &EndpointData{
		Method:          method,
		ServiceName:     svc.Name,
		ServiceVarName:  svc.VarName,
		ServicePkgName:  svc.PkgName,
		Payload:         payload,
		Result:          sds.buildResultData(httpEndpoint, sd),
		Errors:          sds.buildErrorsData(httpEndpoint, sd),
		HeaderSchemes:   hsch,
		BodySchemes:     bosch,
		QuerySchemes:    qsch,
		BasicScheme:     basch,
		Routes:          routes,
		MountHandler:    fmt.Sprintf("Mount%sHandler", method.VarName),
		HandlerInit:     fmt.Sprintf("New%sHandler", method.VarName),
		RequestDecoder:  fmt.Sprintf("Decode%sRequest", method.VarName),
		ResponseEncoder: fmt.Sprintf("Encode%sResponse", method.VarName),
		ErrorEncoder:    fmt.Sprintf("Encode%sError", method.VarName),
		ClientStruct:    "Client",
		EndpointInit:    method.VarName,
		RequestInit:     requestInit,
		HasMixedResults: httpEndpoint.MethodExpr.HasMixedResults(),
		RequestEncoder:  endpointRequestEncoderName(method, payload, basch),
		ResponseDecoder: fmt.Sprintf("Decode%sResponse", method.VarName),
		Requirements:    reqs,
	}
	if httpEndpoint.MethodExpr.IsStreaming() {
		sds.initWebSocketData(endpoint, httpEndpoint, sd)
		initSSEData(endpoint, httpEndpoint, sd)
	}
	sds.initEndpointMultipartData(endpoint, httpEndpoint, method, svc)
	if httpEndpoint.SkipRequestBodyEncodeDecode {
		endpoint.BuildStreamPayload = scope.Unique("Build" + codegen.Goify(method.Name, true) + "StreamPayload")
	}
	if httpEndpoint.Redirect != nil {
		endpoint.Redirect = &RedirectData{
			URL:        httpEndpoint.Redirect.URL,
			StatusCode: statusCodeToHTTPConst(httpEndpoint.Redirect.StatusCode),
		}
	}
	return endpoint
}

func (sds *ServicesData) buildEndpointRoutes(httpEndpoint *expr.HTTPEndpointExpr, method *service.MethodData, svc *service.Data, sd *ServiceData) []*RouteData {
	routesCap := 0
	for _, route := range httpEndpoint.Routes {
		routesCap += len(route.FullPaths())
	}
	routes := make([]*RouteData, 0, routesCap)
	pathCount := 0
	for _, route := range httpEndpoint.Routes {
		for _, path := range route.FullPaths() {
			routes = append(routes, &RouteData{
				Verb:     strings.ToUpper(route.Method),
				Path:     path,
				PathInit: sds.buildPathInitData(httpEndpoint, method, svc, sd, path, pathCount),
			})
			pathCount++
		}
	}
	return routes
}

func (sds *ServicesData) buildPathInitData(httpEndpoint *expr.HTTPEndpointExpr, method *service.MethodData, svc *service.Data, sd *ServiceData, path string, pathCount int) *InitData {
	params := expr.ExtractHTTPWildcards(path)
	initArgs := make([]*InitArgData, len(params))
	pathParamsObj := expr.AsObject(httpEndpoint.PathParams().Type)
	suffix := ""
	if pathCount > 0 {
		suffix = strconv.Itoa(pathCount + 1)
	}
	name := fmt.Sprintf("%s%sPath%s", method.VarName, svc.StructName, suffix)
	for j, arg := range params {
		patt := pathParamsObj.Attribute(arg)
		att := makeHTTPType(patt)
		pointer := httpEndpoint.Params.IsPrimitivePointer(arg, true)
		if expr.IsObject(httpEndpoint.MethodExpr.Payload.Type) {
			pointer = httpEndpoint.MethodExpr.Payload.IsPrimitivePointer(arg, true)
		}
		varName := sd.Scope.Name(codegen.Goify(arg, false))
		validate := ""
		if att.Validation != nil {
			ctx := httpContext(sd.Scope, true, false)
			validate = codegen.AttributeValidationCode(att, nil, ctx, true, expr.IsAlias(att.Type), varName, arg)
		}
		initArgs[j] = &InitArgData{
			Ref: varName,
			AttributeData: &AttributeData{
				Name:        arg,
				VarName:     varName,
				Description: att.Description,
				FieldName:   codegen.Goify(arg, true),
				FieldType:   patt.Type,
				TypeName:    sd.Scope.GoTypeName(att),
				TypeRef:     sd.Scope.GoTypeRef(att),
				Type:        att.Type,
				Pointer:     pointer,
				Required:    true,
				Example:     att.Example(sds.Root.API.ExampleGenerator),
				Validate:    validate,
			},
		}
	}
	var buffer bytes.Buffer
	if err := pathInitTmpl.Execute(&buffer, map[string]any{
		"Args":       initArgs,
		"PathParams": pathParamsObj,
		"PathFormat": expr.HTTPWildcardRegex.ReplaceAllString(path, "/%v"),
	}); err != nil {
		panic(err)
	}
	return &InitData{
		Name:           name,
		Description:    fmt.Sprintf("%s returns the URL path to the %s service %s HTTP endpoint. ", name, svc.Name, method.Name),
		ServerArgs:     initArgs,
		ClientArgs:     initArgs,
		ReturnTypeName: "string",
		ReturnTypeRef:  "string",
		ServerCode:     buffer.String(),
		ClientCode:     buffer.String(),
	}
}

func (sds *ServicesData) buildRequirementSchemes(httpEndpoint *expr.HTTPEndpointExpr) (service.RequirementsData, service.SchemesData, service.SchemesData, service.SchemesData, *service.SchemeData) {
	reqs := make(service.RequirementsData, 0, len(httpEndpoint.Requirements))
	var (
		headerSchemes service.SchemesData
		bodySchemes   service.SchemesData
		querySchemes  service.SchemesData
		basicScheme   *service.SchemeData
	)
	for _, req := range httpEndpoint.Requirements {
		rs := make(service.SchemesData, 0, len(req.Schemes))
		for _, sch := range req.Schemes {
			s := service.BuildSchemeData(sch, httpEndpoint.MethodExpr)
			rs = rs.Append(s)
			switch s.Type {
			case "Basic":
				basicScheme = s
			default:
				switch s.In {
				case "query":
					querySchemes = querySchemes.Append(s)
				case "header":
					headerSchemes = headerSchemes.Append(s)
				default:
					bodySchemes = bodySchemes.Append(s)
				}
			}
		}
		reqs = append(reqs, &service.RequirementData{Schemes: rs, Scopes: req.Scopes})
	}
	return reqs, headerSchemes, bodySchemes, querySchemes, basicScheme
}

func endpointRequestEncoderName(method *service.MethodData, payload *PayloadData, basicScheme *service.SchemeData) string {
	if payload.Request.ClientBody == nil &&
		len(payload.Request.Headers) == 0 &&
		len(payload.Request.QueryParams) == 0 &&
		len(payload.Request.Cookies) == 0 &&
		basicScheme == nil {
		return ""
	}
	return fmt.Sprintf("Encode%sRequest", method.VarName)
}

func (sds *ServicesData) buildClientRequestInit(httpEndpoint *expr.HTTPEndpointExpr, method *service.MethodData, svc *service.Data, routes []*RouteData, sd *ServiceData) *InitData {
	name := fmt.Sprintf("Build%sRequest", method.VarName)
	scope := codegen.NewNameScope()
	scope.Unique("c")
	args := make([]*InitArgData, 0, len(routes[0].PathInit.ClientArgs))
	for _, arg := range routes[0].PathInit.ClientArgs {
		if arg.FieldName == "" {
			continue
		}
		arg.VarName = scope.Unique(arg.VarName)
		arg.Ref = arg.VarName
		_, arg.IsAliased = arg.FieldType.(expr.UserType)
		if arg.IsAliased {
			if svcData := sds.ServicesData.Get(svc.Name); svcData != nil {
				arg.ServiceTypeRef = svcData.Scope.GoTypeRef(&expr.AttributeExpr{Type: arg.Type})
			} else {
				arg.ServiceTypeRef = codegen.Goify(arg.FieldType.Name(), true)
			}
		}
		args = append(args, arg)
	}
	pkg := pkgWithDefault(method.PayloadLoc, svc.PkgName)
	payloadRef := ""
	if len(routes[0].PathInit.ClientArgs) > 0 && httpEndpoint.MethodExpr.Payload.Type != expr.Empty {
		payloadRef = svc.Scope.GoFullTypeRef(httpEndpoint.MethodExpr.Payload, pkg)
	}
	data := map[string]any{
		"PayloadRef":   payloadRef,
		"HasFields":    expr.IsObject(httpEndpoint.MethodExpr.Payload.Type),
		"ServiceName":  svc.Name,
		"EndpointName": method.Name,
		"Args":         args,
		"PathInit":     routes[0].PathInit,
		"Verb":         routes[0].Verb,
		"IsWebSocket":  httpEndpoint.MethodExpr.IsStreaming() && httpEndpoint.SSE == nil,
	}
	if httpEndpoint.SkipRequestBodyEncodeDecode {
		data["RequestStruct"] = pkg + "." + method.RequestStruct
	}
	var buf bytes.Buffer
	if err := requestInitTemplate(sd).Execute(&buf, data); err != nil {
		panic(err) // bug
	}
	return &InitData{
		Name:        name,
		Description: fmt.Sprintf("%s instantiates a HTTP request object with method and path set to call the %q service %q endpoint", name, svc.Name, method.Name),
		ClientCode:  buf.String(),
		ClientArgs:  []*InitArgData{{Ref: "v", AttributeData: &AttributeData{Name: "payload", VarName: "v", TypeRef: "any"}}},
	}
}

func (sds *ServicesData) initEndpointMultipartData(endpoint *EndpointData, httpEndpoint *expr.HTTPEndpointExpr, method *service.MethodData, svc *service.Data) {
	if httpEndpoint.MultipartRequest && !endpoint.Payload.Request.MultipartGenerated {
		endpoint.MultipartRequestDecoder = &MultipartData{
			FuncName:    fmt.Sprintf("%s%sDecoderFunc", svc.StructName, method.VarName),
			InitName:    fmt.Sprintf("New%s%sDecoder", svc.StructName, method.VarName),
			VarName:     fmt.Sprintf("%s%sDecoderFn", svc.VarName, method.VarName),
			ServiceName: svc.Name,
			MethodName:  method.Name,
			Payload:     endpoint.Payload,
		}
	}
	if httpEndpoint.MultipartRequest {
		endpoint.MultipartRequestEncoder = &MultipartData{
			FuncName:    fmt.Sprintf("%s%sEncoderFunc", svc.StructName, method.VarName),
			InitName:    fmt.Sprintf("New%s%sEncoder", svc.StructName, method.VarName),
			VarName:     fmt.Sprintf("%s%sEncoderFn", svc.VarName, method.VarName),
			ServiceName: svc.Name,
			MethodName:  method.Name,
			Payload:     endpoint.Payload,
		}
	}
}

func (sds *ServicesData) collectEndpointBodyAttributeTypes(httpEndpoint *expr.HTTPEndpointExpr, sd *ServiceData) {
	unionBranchTypes := make(map[string]struct{})
	collectUnionBranchUserTypes(httpEndpoint.Body, unionBranchTypes)
	if httpEndpoint.MethodExpr.StreamingPayload.Type != expr.Empty {
		collectUnionBranchUserTypes(httpEndpoint.StreamingBody, unionBranchTypes)
	}

	appendTypeData := func(att *expr.AttributeExpr, req, ptr, server bool, target *[]*TypeData) {
		collectUserTypes(att.Type, func(ut expr.UserType) {
			if d := sds.attributeTypeData(ut, req, ptr, server, sd); d != nil {
				if req && !server && d.ValidateDef == "" {
					if _, ok := unionBranchTypes[ut.ID()]; ok {
						d.ValidateDef = "// no validations"
						d.ValidateRef = fmt.Sprintf("err = Validate%s(v)", d.VarName)
					}
				}
				*target = append(*target, d)
			}
		})
	}
	appendTypeData(httpEndpoint.Body, true, true, true, &sd.ServerBodyAttributeTypes)
	appendTypeData(httpEndpoint.Body, true, false, false, &sd.ClientBodyAttributeTypes)

	if httpEndpoint.MethodExpr.StreamingPayload.Type != expr.Empty {
		appendTypeData(httpEndpoint.StreamingBody, true, true, true, &sd.ServerBodyAttributeTypes)
		appendTypeData(httpEndpoint.StreamingBody, true, false, false, &sd.ClientBodyAttributeTypes)
	}

	if httpEndpoint.MethodExpr.Result != nil {
		for _, response := range httpEndpoint.Responses {
			collectUserTypes(response.Body.Type, func(ut expr.UserType) {
				if d := sds.attributeTypeData(ut, false, true, false, sd); d != nil {
					sd.ClientBodyAttributeTypes = append(sd.ClientBodyAttributeTypes, d)
				}
			})
		}
	}
	for _, httpError := range httpEndpoint.HTTPErrors {
		collectUserTypes(httpError.Response.Body.Type, func(ut expr.UserType) {
			if d := sds.attributeTypeData(ut, false, true, false, sd); d != nil {
				sd.ClientBodyAttributeTypes = append(sd.ClientBodyAttributeTypes, d)
			}
		})
	}
}

func (sds *ServicesData) collectEndpointUnionTypes(httpSvc *expr.HTTPServiceExpr, scope *codegen.NameScope) []*service.UnionTypeData {
	unionByHash := make(map[string]*service.UnionTypeData)
	seenUnionTypes := make(map[string]struct{})
	for _, endpoint := range httpSvc.HTTPEndpoints {
		collectHTTPUnionTypes(endpoint.Body, scope, unionByHash, seenUnionTypes)
		if endpoint.MethodExpr.StreamingPayload.Type != expr.Empty {
			collectHTTPUnionTypes(endpoint.StreamingBody, scope, unionByHash, seenUnionTypes)
		}
		if endpoint.MethodExpr.Result != nil {
			for _, response := range endpoint.Responses {
				collectHTTPUnionTypes(response.Body, scope, unionByHash, seenUnionTypes)
			}
		}
		for _, httpError := range endpoint.HTTPErrors {
			collectHTTPUnionTypes(httpError.Response.Body, scope, unionByHash, seenUnionTypes)
		}
	}
	unions := make([]*service.UnionTypeData, 0, len(unionByHash))
	for _, union := range unionByHash {
		unions = append(unions, union)
	}
	sort.Slice(unions, func(i, j int) bool {
		return unions[i].Name < unions[j].Name
	})
	return unions
}

// requestInitTemplate returns the template used to render request constructors.
func requestInitTemplate(svcData *ServiceData) *template.Template {
	return template.Must(
		template.New("request-init").
			Funcs(template.FuncMap{
				"goTypeRef": func(dt expr.DataType, svc string) string {
					return svcData.Scope.GoTypeRef(&expr.AttributeExpr{Type: dt})
				},
				"isAliased": func(dt expr.DataType) bool {
					_, ok := dt.(expr.UserType)
					return ok
				},
				"isWebSocketEndpoint": IsWebSocketEndpoint,
			}).
			Parse(httpTemplates.Read(requestInitT)),
	)
}

// makeHTTPType traverses the attribute recursively and performs these actions:
//
// * removes aliased user type by replacing them with the underlying type.
// * changes unions into structs with Type and Value fields.
func makeHTTPType(att *expr.AttributeExpr) *expr.AttributeExpr {
	if att == nil {
		return nil
	}
	att = expr.DupAtt(att)
	return makeHTTPTypeRecursive(att, make(map[string]struct{}))
}

func makeHTTPTypeRecursive(att *expr.AttributeExpr, seen map[string]struct{}) *expr.AttributeExpr {
	switch dt := att.Type.(type) {
	case expr.UserType:
		if _, ok := dt.(*expr.ResultTypeExpr); !ok && !expr.IsObject(dt) {
			// Aliased user type. Use the underlying aliased type instead of
			// generating new types in the client and server packages
			att.Type = dt.Attribute().Type
			if v := dt.Attribute().Validation; v != nil {
				if att.Validation == nil {
					att.Validation = v
				} else {
					att.Validation.Merge(v)
				}
			}
			att.DefaultValue = dt.Attribute().DefaultValue
			att.UserExamples = dt.Attribute().UserExamples
		}
		if _, ok := seen[dt.ID()]; ok {
			return att
		}
		seen[dt.ID()] = struct{}{}
		dt.SetAttribute(makeHTTPTypeRecursive(dt.Attribute(), seen))
	case *expr.Array:
		dt.ElemType = makeHTTPTypeRecursive(dt.ElemType, seen)
	case *expr.Map:
		dt.KeyType = makeHTTPTypeRecursive(dt.KeyType, seen)
		dt.ElemType = makeHTTPTypeRecursive(dt.ElemType, seen)
	case *expr.Object:
		obj := make(expr.Object, len(*dt))
		for i, nat := range *dt {
			obj[i] = &expr.NamedAttributeExpr{Name: nat.Name, Attribute: makeHTTPTypeRecursive(nat.Attribute, seen)}
		}
		att.Type = &obj
	case *expr.Union:
		// Unions are represented as first-class sum types; HTTP uses the same
		// type for request and response bodies.
	}
	return att
}

func generatedMultipartRequestData(e *expr.HTTPEndpointExpr) (bool, []*MultipartFileFieldData) {
	if !e.MultipartRequest || e.Body == nil || e.Body.Type == expr.Empty || !expr.IsObject(e.Body.Type) {
		return false, nil
	}
	if !supportsGeneratedMultipartObject(e.Body) {
		return false, nil
	}
	fileFields := multipartFileFields(e.Body)
	if len(fileFields) == 1 {
		bodyObj := expr.AsObject(e.Body.Type)
		if attr := bodyObj.Attribute("filename"); attr != nil && attr.Type.Kind() == expr.StringKind {
			fileFields[0].PopulateFilename = true
		}
		if attr := bodyObj.Attribute("content_type"); attr != nil && attr.Type.Kind() == expr.StringKind {
			fileFields[0].PopulateContentType = true
		}
	}
	return true, fileFields
}

func supportsGeneratedMultipartObject(body *expr.AttributeExpr) bool {
	obj := expr.AsObject(body.Type)
	if obj == nil {
		return false
	}
	for _, nat := range *obj {
		if nat.Attribute.Type == expr.Bytes {
			continue
		}
		if !supportsGeneratedMultipartNested(nat.Attribute) {
			return false
		}
	}
	return true
}

func supportsGeneratedMultipartNested(att *expr.AttributeExpr) bool {
	switch actual := att.Type.(type) {
	case expr.Primitive:
		return actual != expr.Any && actual != expr.Bytes
	case expr.UserType:
		return supportsGeneratedMultipartNested(actual.Attribute())
	case *expr.Object:
		for _, nat := range *actual {
			if !supportsGeneratedMultipartNested(nat.Attribute) {
				return false
			}
		}
		return true
	case *expr.Map:
		if !isSupportedMultipartScalar(actual.KeyType.Type) {
			return false
		}
		return supportsGeneratedMultipartNested(actual.ElemType)
	case *expr.Array:
		return supportsGeneratedMultipartCollectionElem(actual.ElemType)
	case *expr.Union:
		return false
	default:
		return false
	}
}

func supportsGeneratedMultipartCollectionElem(att *expr.AttributeExpr) bool {
	switch actual := att.Type.(type) {
	case expr.Primitive:
		return actual != expr.Any && actual != expr.Bytes
	case expr.UserType:
		return supportsGeneratedMultipartCollectionElem(actual.Attribute())
	case *expr.Object:
		for _, nat := range *actual {
			if !supportsGeneratedMultipartNested(nat.Attribute) {
				return false
			}
		}
		return true
	case *expr.Map:
		if !isSupportedMultipartScalar(actual.KeyType.Type) {
			return false
		}
		return supportsGeneratedMultipartCollectionElem(actual.ElemType)
	case *expr.Array:
		return supportsGeneratedMultipartCollectionElem(actual.ElemType)
	default:
		return false
	}
}

func isSupportedMultipartScalar(dt expr.DataType) bool {
	prim, ok := dt.(expr.Primitive)
	return ok && prim != expr.Any && prim != expr.Bytes
}

func multipartFileFields(body *expr.AttributeExpr) []*MultipartFileFieldData {
	obj := expr.AsObject(body.Type)
	if obj == nil {
		return nil
	}
	fields := make([]*MultipartFileFieldData, 0, len(*obj))
	for _, nat := range *obj {
		if nat.Attribute.Type != expr.Bytes {
			continue
		}
		name := strings.Split(nat.Name, ":")[0]
		fields = append(fields, &MultipartFileFieldData{
			Name:     name,
			HTTPName: name,
			Required: body.IsRequired(name),
		})
	}
	return fields
}

// buildPayloadData returns the data structure used to describe the endpoint
// payload including the HTTP request details. It also returns the user types
// used by the request body type recursively if any.
func (sds *ServicesData) buildPayloadData(e *expr.HTTPEndpointExpr, sd *ServiceData) *PayloadData {
	e.Body = makeHTTPType(e.Body)
	var (
		payload    = e.MethodExpr.Payload
		svc        = sd.Service
		body       = e.Body.Type
		ep         = svc.Method(e.MethodExpr.Name)
		httpsvrctx = httpContext(sd.Scope, true, true)
		httpclictx = httpContext(sd.Scope, true, false)
		pkg        = pkgWithDefault(ep.PayloadLoc, svc.PkgName)
		svcctx     = serviceContext(pkg, sd.Service.Scope)

		request       *RequestData
		mapQueryParam *ParamData
	)
	request, mapQueryParam = sds.buildPayloadRequestData(e, payload, svcctx, httpsvrctx, sd)

	var init *InitData
	if needInit(payload.Type) {
		init = sds.buildPayloadInitData(e, payload, body, ep, pkg, request, svcctx, httpsvrctx, httpclictx, sd)
	}
	request.PayloadInit = init

	var (
		returnValue string
		name        string
		ref         string
	)
	if payload.Type != expr.Empty {
		name = svc.Scope.GoFullTypeName(payload, pkg)
		ref = svc.Scope.GoFullTypeRef(payload, pkg)
	}
	if init == nil {
		if o := expr.AsObject(e.Params.Type); o != nil && len(*o) > 0 {
			returnValue = codegen.Goify((*o)[0].Name, false)
		} else if o := expr.AsObject(e.Headers.Type); o != nil && len(*o) > 0 {
			returnValue = codegen.Goify((*o)[0].Name, false)
		} else if o := expr.AsObject(e.Cookies.Type); o != nil && len(*o) > 0 {
			returnValue = codegen.Goify((*o)[0].Name, false)
		} else if e.MapQueryParams != nil && *e.MapQueryParams == "" {
			returnValue = mapQueryParam.VarName
		}
	}
	data := &PayloadData{
		Name:               name,
		Ref:                ref,
		Request:            request,
		DecoderReturnValue: returnValue,
	}
	if e.IsJSONRPC() {
		obj := expr.AsObject(e.MethodExpr.Payload.Type)
		if obj != nil {
			for _, att := range *obj {
				if _, ok := att.Attribute.Meta["jsonrpc:id"]; ok {
					data.IDAttribute = codegen.Goify(att.Name, true)
					data.IDAttributeRequired = e.MethodExpr.Payload.IsRequired(att.Name)
					break
				}
			}
		}
	}
	return data
}

func (sds *ServicesData) buildPayloadRequestData(e *expr.HTTPEndpointExpr, payload *expr.AttributeExpr, svcctx, httpsvrctx *codegen.AttributeContext, sd *ServiceData) (*RequestData, *ParamData) {
	serverBodyData := sds.buildRequestBodyType(e.Body, payload, e, true, sd)
	clientBodyData := sds.buildRequestBodyType(e.Body, payload, e, false, sd)
	paramsData := sds.extractPathParams(e.PathParams(), payload, sd.Scope)
	queryData := sds.extractQueryParams(e.QueryParams(), payload, sd.Scope)
	headersData := sds.extractHeaders(e.Headers, payload, svcctx, sd.Scope)
	cookiesData := sds.extractCookies(e.Cookies, payload, svcctx, sd.Scope)
	multipartGen, multipartFiles := generatedMultipartRequestData(e)
	mapQueryParam := sds.buildMapQueryParam(e, payload, httpsvrctx, sd)
	if mapQueryParam != nil {
		queryData = append(queryData, mapQueryParam)
	}
	if serverBodyData != nil {
		sd.ServerTypeNames[serverBodyData.Name] = false
		sd.ClientTypeNames[serverBodyData.Name] = false
	}
	origin := ""
	mustHaveBody := true
	if e.Body.Type != expr.Empty {
		if e.OptionalRequestBody {
			mustHaveBody = false
		}
		if o, ok := e.Body.Meta["origin:attribute"]; ok {
			origin = o[0]
			if !payload.IsRequired(o[0]) {
				mustHaveBody = false
			}
		}
	}
	return &RequestData{
		PathParams:          paramsData,
		QueryParams:         queryData,
		Headers:             headersData,
		Cookies:             cookiesData,
		ServerBody:          serverBodyData,
		ClientBody:          clientBodyData,
		PayloadAttr:         codegen.Goify(origin, true),
		PayloadType:         e.MethodExpr.Payload.Type,
		MustHaveBody:        mustHaveBody,
		MustValidate:        payloadRequestNeedsValidation(paramsData, queryData, headersData, cookiesData),
		Multipart:           e.MultipartRequest,
		MultipartGenerated:  multipartGen,
		MultipartFileFields: multipartFiles,
		FormEncoded:         e.FormRequest,
	}, mapQueryParam
}

func (sds *ServicesData) buildMapQueryParam(e *expr.HTTPEndpointExpr, payload *expr.AttributeExpr, httpsvrctx *codegen.AttributeContext, sd *ServiceData) *ParamData {
	if e.MapQueryParams == nil {
		return nil
	}
	fieldName := ""
	name := "query"
	required := true
	pAtt := payload
	if n := *e.MapQueryParams; n != "" {
		pAtt = expr.AsObject(payload.Type).Attribute(n)
		required = payload.IsRequired(n)
		name = n
		fieldName = codegen.Goify(name, true)
	}
	varName := codegen.Goify(name, false)
	return &ParamData{
		MapQueryParams: e.MapQueryParams,
		Map:            expr.AsMap(payload.Type) != nil,
		Element: &Element{
			HTTPName: name,
			AttributeData: &AttributeData{
				Name:         name,
				VarName:      varName,
				FieldName:    fieldName,
				FieldType:    pAtt.Type,
				Required:     required,
				Type:         pAtt.Type,
				TypeName:     sd.Scope.GoTypeName(pAtt),
				TypeRef:      sd.Scope.GoTypeRef(pAtt),
				Validate:     codegen.AttributeValidationCode(pAtt, nil, httpsvrctx, required, expr.IsAlias(pAtt.Type), varName, name),
				DefaultValue: pAtt.DefaultValue,
				Example:      pAtt.Example(sds.Root.API.ExampleGenerator),
			},
		},
	}
}

func payloadRequestNeedsValidation(paramsData []*ParamData, queryData []*ParamData, headersData []*HeaderData, cookiesData []*CookieData) bool {
	for _, cookie := range cookiesData {
		if cookie.Required || cookie.Validate != "" || needConversion(cookie.Type) {
			return true
		}
	}
	for _, param := range paramsData {
		if param.Validate != "" || needConversion(param.Type) {
			return true
		}
	}
	for _, query := range queryData {
		if query.Map || query.Validate != "" || query.Required || needConversion(query.Type) {
			return true
		}
	}
	for _, header := range headersData {
		if header.Validate != "" || header.Required || needConversion(header.Type) {
			return true
		}
	}
	return false
}

func (sds *ServicesData) buildPayloadInitData(
	e *expr.HTTPEndpointExpr,
	payload *expr.AttributeExpr,
	body expr.DataType,
	ep *service.MethodData,
	pkg string,
	request *RequestData,
	svcctx, httpsvrctx, httpclictx *codegen.AttributeContext,
	sd *ServiceData,
) *InitData {
	svc := sd.Service
	argsCap := len(request.PathParams) + len(request.QueryParams) + len(request.Headers) + len(request.Cookies)
	n := codegen.Goify(ep.Name, true)
	p := codegen.Goify(ep.Payload, true)
	name := ""
	if strings.HasPrefix(p, n) {
		p = svc.Scope.HashedUnique(payload.Type, p)
		name = fmt.Sprintf("New%s", p)
	} else {
		name = fmt.Sprintf("New%s%s", n, p)
	}
	serverArgs, clientArgs := sds.buildPayloadBodyArgs(e, body, argsCap, httpclictx, httpsvrctx, sd)
	args := buildPayloadFieldArgs(request)
	serverArgs = append(serverArgs, args...)
	clientArgs = append(clientArgs, args...)
	serverCode, clientCode, origin, pointer := sds.buildPayloadTransformCode(e, payload, body, svcctx, httpsvrctx, httpclictx, sd)
	return &InitData{
		Name:                     name,
		Description:              fmt.Sprintf("%s builds a %s service %s endpoint payload.", name, svc.Name, e.Name()),
		ServerArgs:               serverArgs,
		ClientArgs:               clientArgs,
		CLIArgs:                  buildBasicAuthCLIArgs(ep, e, svc, httpsvrctx, sds.Root.API.ExampleGenerator),
		ReturnTypeName:           svc.Scope.GoFullTypeName(payload, pkg),
		ReturnTypeRef:            svc.Scope.GoFullTypeRef(payload, pkg),
		ReturnIsStruct:           expr.IsObject(payload.Type),
		ReturnTypeAttribute:      codegen.Goify(origin, true),
		ReturnTypePkg:            pkg,
		ServerCode:               serverCode,
		ClientCode:               clientCode,
		ReturnIsPrimitivePointer: pointer,
	}
}

func (sds *ServicesData) buildPayloadBodyArgs(
	e *expr.HTTPEndpointExpr,
	body expr.DataType,
	argsCap int,
	httpclictx, httpsvrctx *codegen.AttributeContext,
	sd *ServiceData,
) ([]*InitArgData, []*InitArgData) {
	serverArgs := make([]*InitArgData, 0, argsCap+1)
	clientArgs := make([]*InitArgData, 0, argsCap+1)
	if body == expr.Empty {
		return serverArgs, clientArgs
	}
	svcode := ""
	cvcode := ""
	if ut, ok := body.(expr.UserType); ok {
		if val := ut.Attribute().Validation; val != nil {
			svcode = codegen.ValidationCode(ut.Attribute(), ut, httpsvrctx, true, expr.IsAlias(ut), false, "body")
			cvcode = codegen.ValidationCode(ut.Attribute(), ut, httpclictx, true, expr.IsAlias(ut), false, "body")
		}
	}
	serverArgs = append(serverArgs, &InitArgData{
		Ref: sd.Scope.GoVar("body", body),
		AttributeData: &AttributeData{
			Name:     "body",
			VarName:  "body",
			TypeName: sd.Scope.GoTypeName(e.Body),
			TypeRef:  sd.Scope.GoTypeRef(e.Body),
			Type:     body,
			Required: true,
			Example:  e.Body.Example(sds.Root.API.ExampleGenerator),
			Validate: svcode,
		},
	})
	clientArgs = append(clientArgs, &InitArgData{
		Ref: sd.Scope.GoVar("body", body),
		AttributeData: &AttributeData{
			Name:     "body",
			VarName:  "body",
			TypeName: sd.Scope.GoTypeNameWithDefaults(e.Body),
			TypeRef:  sd.Scope.GoTypeRefWithDefaults(e.Body),
			Type:     body,
			Required: true,
			Example:  e.Body.Example(sds.Root.API.ExampleGenerator),
			Validate: cvcode,
		},
	})
	return serverArgs, clientArgs
}

func buildPayloadFieldArgs(request *RequestData) []*InitArgData {
	args := make([]*InitArgData, 0, len(request.PathParams)+len(request.QueryParams)+len(request.Headers)+len(request.Cookies))
	appendField := func(
		ref string,
		name string,
		varName string,
		description string,
		fieldName string,
		fieldPointer bool,
		fieldType expr.DataType,
		typeName string,
		typeRef string,
		typ expr.DataType,
		pointer bool,
		required bool,
		defaultValue any,
		validate string,
		example any,
	) {
		args = append(args, &InitArgData{
			Ref: ref,
			AttributeData: &AttributeData{
				Name:         name,
				VarName:      varName,
				Description:  description,
				FieldName:    fieldName,
				FieldPointer: fieldPointer,
				FieldType:    fieldType,
				TypeName:     typeName,
				TypeRef:      typeRef,
				Type:         typ,
				Pointer:      pointer,
				Required:     required,
				DefaultValue: defaultValue,
				Validate:     validate,
				Example:      example,
			},
		})
	}
	for _, param := range request.PathParams {
		appendField(param.VarName, param.Name, param.VarName, param.Description, param.FieldName, param.FieldPointer, param.FieldType, param.TypeName, param.TypeRef, param.Type, param.Pointer, param.Required, nil, param.Validate, param.Example)
	}
	for _, param := range request.QueryParams {
		appendField(param.VarName, param.Name, param.VarName, "", param.FieldName, param.FieldPointer, param.FieldType, param.TypeName, param.TypeRef, param.Type, param.Pointer, param.Required, param.DefaultValue, param.Validate, param.Example)
	}
	for _, header := range request.Headers {
		appendField(header.VarName, header.Name, header.VarName, "", header.FieldName, header.FieldPointer, header.FieldType, header.TypeName, header.TypeRef, header.Type, header.Pointer, header.Required, header.DefaultValue, header.Validate, header.Example)
	}
	for _, cookie := range request.Cookies {
		appendField(cookie.VarName, cookie.Name, cookie.VarName, "", cookie.FieldName, cookie.FieldPointer, cookie.FieldType, cookie.TypeName, cookie.TypeRef, cookie.Type, cookie.Pointer, cookie.Required, cookie.DefaultValue, cookie.Validate, cookie.Example)
	}
	return args
}

func buildBasicAuthCLIArgs(ep *service.MethodData, e *expr.HTTPEndpointExpr, svc *service.Data, httpsvrctx *codegen.AttributeContext, generator *expr.ExampleGenerator) []*InitArgData {
	for _, requirement := range ep.Requirements {
		for _, scheme := range requirement.Schemes {
			if scheme.Type != "Basic" {
				continue
			}
			uatt := e.MethodExpr.Payload.Find(scheme.UsernameAttr)
			uref := svc.Scope.GoTypeRef(uatt)
			if scheme.UsernamePointer {
				uref = "*" + uref
			}
			patt := e.MethodExpr.Payload.Find(scheme.PasswordAttr)
			pref := svc.Scope.GoTypeRef(patt)
			if scheme.PasswordPointer {
				pref = "*" + pref
			}
			return []*InitArgData{
				{
					Ref: scheme.UsernameAttr,
					AttributeData: &AttributeData{
						Name:         scheme.UsernameAttr,
						VarName:      scheme.UsernameAttr,
						FieldName:    scheme.UsernameField,
						FieldPointer: scheme.UsernamePointer,
						FieldType:    uatt.Type,
						Description:  uatt.Description,
						Required:     scheme.UsernameRequired,
						TypeName:     svc.Scope.GoTypeName(uatt),
						TypeRef:      uref,
						Type:         uatt.Type,
						Pointer:      scheme.UsernamePointer,
						Validate:     codegen.ValidationCode(uatt, nil, httpsvrctx, scheme.UsernameRequired, expr.IsAlias(uatt.Type), false, scheme.UsernameAttr),
						Example:      uatt.Example(generator),
					},
				},
				{
					Ref: scheme.PasswordAttr,
					AttributeData: &AttributeData{
						Name:         scheme.PasswordAttr,
						VarName:      scheme.PasswordAttr,
						FieldName:    scheme.PasswordField,
						FieldPointer: scheme.PasswordPointer,
						FieldType:    patt.Type,
						Description:  patt.Description,
						Required:     scheme.PasswordRequired,
						TypeName:     svc.Scope.GoTypeName(patt),
						TypeRef:      pref,
						Type:         patt.Type,
						Pointer:      scheme.PasswordPointer,
						Validate:     codegen.ValidationCode(patt, nil, httpsvrctx, scheme.PasswordRequired, expr.IsAlias(patt.Type), false, scheme.PasswordAttr),
						Example:      patt.Example(generator),
					},
				},
			}
		}
	}
	return nil
}

func buildBodyInitArg(scope *codegen.NameScope, body *expr.AttributeExpr, addressObject bool) *InitArgData {
	ref := "body"
	if addressObject && expr.IsObject(body.Type) {
		ref = "&body"
	}
	return &InitArgData{
		Ref: ref,
		AttributeData: &AttributeData{
			Name:    "body",
			VarName: "body",
			TypeRef: scope.GoTypeRef(body),
		},
	}
}

func buildHeaderInitArgs(headers []*HeaderData) []*InitArgData {
	args := make([]*InitArgData, 0, len(headers))
	for _, header := range headers {
		args = append(args, &InitArgData{
			Ref: header.VarName,
			AttributeData: &AttributeData{
				Name:         header.Name,
				VarName:      header.VarName,
				FieldName:    header.FieldName,
				FieldPointer: header.FieldPointer,
				FieldType:    header.FieldType,
				Required:     header.Required,
				Pointer:      header.Pointer,
				TypeRef:      header.TypeRef,
				Type:         header.Type,
				Validate:     header.Validate,
				Example:      header.Example,
			},
		})
	}
	return args
}

func buildCookieInitArgs(cookies []*CookieData) []*InitArgData {
	args := make([]*InitArgData, 0, len(cookies))
	for _, cookie := range cookies {
		args = append(args, &InitArgData{
			Ref: cookie.VarName,
			AttributeData: &AttributeData{
				Name:         cookie.Name,
				VarName:      cookie.VarName,
				FieldName:    cookie.FieldName,
				FieldPointer: cookie.FieldPointer,
				FieldType:    cookie.FieldType,
				Required:     cookie.Required,
				Pointer:      cookie.Pointer,
				TypeRef:      cookie.TypeRef,
				Type:         cookie.Type,
				Validate:     cookie.Validate,
				Example:      cookie.Example,
			},
		})
	}
	return args
}

func responseFieldsNeedValidation(headers []*HeaderData, cookies []*CookieData) bool {
	for _, header := range headers {
		if header.Validate != "" || header.Required || needConversion(header.Type) {
			return true
		}
	}
	for _, cookie := range cookies {
		if cookie.Validate != "" || cookie.Required || needConversion(cookie.Type) {
			return true
		}
	}
	return false
}

func (sds *ServicesData) buildPayloadTransformCode(
	e *expr.HTTPEndpointExpr,
	payload *expr.AttributeExpr,
	body expr.DataType,
	svcctx, httpsvrctx, httpclictx *codegen.AttributeContext,
	sd *ServiceData,
) (string, string, string, bool) {
	serverCode := ""
	clientCode := ""
	origin := ""
	pointer := false
	var err error
	pAtt := payload
	if body != expr.Empty {
		if o, ok := e.Body.Meta["origin:attribute"]; ok {
			origin = o[0]
			pAtt = expr.AsObject(payload.Type).Attribute(origin)
			pointer = !payload.IsRequired(o[0]) && expr.IsPrimitive(pAtt.Type)
		}
		var helpers []*codegen.TransformFunctionData
		serverCode, helpers, err = unmarshal(e.Body, pAtt, "body", httpsvrctx, svcctx)
		if err == nil {
			sd.ServerTransformHelpers = codegen.AppendHelpers(sd.ServerTransformHelpers, helpers)
		}
		clientCode, helpers, err = marshal(e.Body, pAtt, "body", "v", httpclictx, svcctx)
		if err == nil {
			sd.ClientTransformHelpers = codegen.AppendHelpers(sd.ClientTransformHelpers, helpers)
		}
	} else if expr.IsArray(payload.Type) || expr.IsMap(payload.Type) {
		if params := expr.AsObject(e.Params.Type); len(*params) > 0 {
			var helpers []*codegen.TransformFunctionData
			source := codegen.Goify((*params)[0].Name, false)
			serverCode, helpers, err = unmarshal((*params)[0].Attribute, payload, source, httpsvrctx, svcctx)
			if err == nil {
				sd.ServerTransformHelpers = codegen.AppendHelpers(sd.ServerTransformHelpers, helpers)
			}
			clientCode, helpers, err = marshal((*params)[0].Attribute, payload, source, "v", httpclictx, svcctx)
			if err == nil {
				sd.ClientTransformHelpers = codegen.AppendHelpers(sd.ClientTransformHelpers, helpers)
			}
		}
	}
	if err != nil {
		fmt.Println(err.Error()) // TBD validate DSL so errors are not possible
	}
	return serverCode, clientCode, origin, pointer
}

// buildResultData builds the result data for the given service endpoint.
func (sds *ServicesData) buildResultData(e *expr.HTTPEndpointExpr, sd *ServiceData) *ResultData {
	var (
		svc    = sd.Service
		ep     = svc.Method(e.MethodExpr.Name)
		pkg    = pkgWithDefault(ep.ResultLoc, svc.PkgName)
		result = e.MethodExpr.Result

		name string
		ref  string
		view string
	)

	view = expr.DefaultView
	if v, ok := result.Meta.Last(expr.ViewMetaKey); ok {
		view = v
	}
	if result.Type != expr.Empty {
		name = svc.Scope.GoFullTypeName(result, pkg)
		ref = svc.Scope.GoFullTypeRef(result, pkg)
	}

	var (
		mustInit  bool
		responses []*ResponseData
	)
	{
		viewed := false
		if ep.ViewedResult != nil {
			result = expr.AsObject(ep.ViewedResult.Type).Attribute("projected")
			viewed = true
		}
		responses = sds.buildResponses(e, result, viewed, sd)
		for _, r := range responses {
			// response has a body, headers, cookies or tag
			if len(r.ServerBody) > 0 || len(r.Headers) > 0 || len(r.Cookies) > 0 || r.TagName != "" {
				mustInit = true
			}
		}
	}
	idAtt := ""
	idAttRequired := false
	if e.IsJSONRPC() && result.Type != expr.Empty {
		obj := expr.AsObject(result.Type)
		if obj != nil {
			for _, att := range *obj {
				if _, ok := att.Attribute.Meta["jsonrpc:id"]; ok {
					idAtt = codegen.Goify(att.Name, true)
					idAttRequired = result.IsRequired(att.Name)
					break
				}
			}
		}
	}
	return &ResultData{
		IsStruct:            expr.IsObject(result.Type),
		Name:                name,
		Ref:                 ref,
		IDAttribute:         idAtt,
		IDAttributeRequired: idAttRequired,
		Responses:           responses,
		View:                view,
		MustInit:            mustInit,
	}
}

// buildResponses builds the response data for all the responses in the endpoint
// expression. The response headers, cookies and body for each response are
// inferred from the method's result expression if not specified explicitly.
//
// viewed parameter indicates if the method result uses views.
func (sds *ServicesData) buildResponses(e *expr.HTTPEndpointExpr, result *expr.AttributeExpr, viewed bool, sd *ServiceData) []*ResponseData {
	var (
		responses []*ResponseData

		svc        = sd.Service
		md         = svc.Method(e.Name())
		pkg        = pkgWithDefault(md.ResultLoc, svc.PkgName)
		httpclictx = httpContext(sd.Scope, false, false)
		scope      = svc.Scope
		svcctx     = serviceContext(pkg, sd.Service.Scope)
	)
	{
		if viewed {
			scope = svc.ViewScope
			svcctx = viewContext(sd.Service.ViewsPkg, sd.Service.ViewScope)
		}
		notag := -1
		for i, resp := range e.Responses {
			resp.Body = expr.DupAtt(resp.Body)
			resp.Body = makeHTTPType(resp.Body)
			if resp.Tag[0] == "" {
				if notag > -1 {
					continue // we don't want more than one response with no tag
				}
				notag = i
			}
			responses = append(responses, sds.buildSingleResponseData(e, resp, result, viewed, md, pkg, httpclictx, scope, svcctx, sd))
		}
		count := len(responses)
		if notag >= 0 && notag < count-1 {
			// Make sure tagless response is last
			responses[notag], responses[count-1] = responses[count-1], responses[notag]
		}
	}
	return responses
}

func (sds *ServicesData) buildSingleResponseData(
	e *expr.HTTPEndpointExpr,
	resp *expr.HTTPResponseExpr,
	result *expr.AttributeExpr,
	viewed bool,
	md *service.MethodData,
	pkg string,
	httpclictx *codegen.AttributeContext,
	scope *codegen.NameScope,
	svcctx *codegen.AttributeContext,
	sd *ServiceData,
) *ResponseData {
	headersData := sds.extractHeaders(resp.Headers, result, svcctx, scope)
	cookiesData := sds.extractResponseCookies(resp.Cookies, result, svcctx, scope)
	origin, resAttr := responseOriginAttribute(resp, result)
	serverBodyData, clientBodyData := sds.buildResponseBodyData(resp, result, origin, viewed, md, e, sd)
	init := sds.buildResponseResultInit(resp, result, resAttr, origin, viewed, md, pkg, httpclictx, scope, svcctx, headersData, cookiesData, e, sd)
	tagName, tagValue, tagPointer := responseTagData(resp, result, viewed)
	return &ResponseData{
		StatusCode:   statusCodeToHTTPConst(resp.StatusCode),
		Description:  resp.Description,
		Headers:      headersData,
		Cookies:      cookiesData,
		ContentType:  resp.ContentType,
		ServerBody:   serverBodyData,
		ClientBody:   clientBodyData,
		ResultInit:   init,
		TagName:      tagName,
		TagValue:     tagValue,
		TagPointer:   tagPointer,
		MustValidate: responseFieldsNeedValidation(headersData, cookiesData),
		ResultAttr:   codegen.Goify(origin, true),
		ViewedResult: md.ViewedResult,
	}
}

func responseOriginAttribute(resp *expr.HTTPResponseExpr, result *expr.AttributeExpr) (string, *expr.AttributeExpr) {
	if resp.Body.Type == expr.Empty {
		return "", result
	}
	if origin, ok := resp.Body.Meta["origin:attribute"]; ok {
		return origin[0], expr.AsObject(result.Type).Attribute(origin[0])
	}
	return "", result
}

func (sds *ServicesData) buildResponseBodyData(
	resp *expr.HTTPResponseExpr,
	result *expr.AttributeExpr,
	origin string,
	viewed bool,
	md *service.MethodData,
	e *expr.HTTPEndpointExpr,
	sd *ServiceData,
) ([]*TypeData, *TypeData) {
	var serverBodyData []*TypeData
	var clientBodyData *TypeData
	if viewed {
		vname := ""
		if origin != "" {
			if sbd := sds.buildResponseBodyType(resp.Body, result, md.ResultLoc, e, true, &vname, sd); sbd != nil {
				serverBodyData = append(serverBodyData, sbd)
			}
		} else if v, ok := e.MethodExpr.Result.Meta.Last(expr.ViewMetaKey); ok {
			if sbd := sds.buildResponseBodyType(resp.Body, result, md.ResultLoc, e, true, &v, sd); sbd != nil {
				serverBodyData = append(serverBodyData, sbd)
			}
		} else {
			for _, view := range md.ViewedResult.Views {
				if sbd := sds.buildResponseBodyType(resp.Body, result, md.ResultLoc, e, true, &view.Name, sd); sbd != nil {
					serverBodyData = append(serverBodyData, sbd)
				}
			}
		}
		clientBodyData = sds.buildResponseBodyType(resp.Body, result, md.ResultLoc, e, false, &vname, sd)
	} else {
		if sbd := sds.buildResponseBodyType(resp.Body, result, md.ResultLoc, e, true, nil, sd); sbd != nil {
			serverBodyData = append(serverBodyData, sbd)
		}
		clientBodyData = sds.buildResponseBodyType(resp.Body, result, md.ResultLoc, e, false, nil, sd)
	}
	if clientBodyData != nil && clientBodyData.Def != "" {
		sd.ClientTypeNames[clientBodyData.Name] = false
	}
	return serverBodyData, clientBodyData
}

func (sds *ServicesData) buildResponseResultInit(
	resp *expr.HTTPResponseExpr,
	result *expr.AttributeExpr,
	resAttr *expr.AttributeExpr,
	origin string,
	viewed bool,
	md *service.MethodData,
	pkg string,
	httpclictx *codegen.AttributeContext,
	scope *codegen.NameScope,
	svcctx *codegen.AttributeContext,
	headersData []*HeaderData,
	cookiesData []*CookieData,
	e *expr.HTTPEndpointExpr,
	sd *ServiceData,
) *InitData {
	if !needInit(result.Type) {
		return nil
	}
	tname := sd.Service.Scope.GoFullTypeName(result, pkg)
	tref := sd.Service.Scope.GoFullTypeRef(result, pkg)
	if viewed {
		tname = sd.Service.ViewScope.GoFullTypeName(result, sd.Service.ViewsPkg)
		tref = sd.Service.ViewScope.GoFullTypeRef(result, sd.Service.ViewsPkg)
	}
	status := codegen.Goify(http.StatusText(resp.StatusCode), true)
	n := codegen.Goify(md.Name, true)
	r := codegen.Goify(md.Result, true)
	if strings.HasPrefix(r, n) {
		r = scope.HashedUnique(result.Type, r)
	}
	name := fmt.Sprintf("New%s%s%s", n, r, status)
	if strings.HasPrefix(codegen.Goify(md.Result, true), n) {
		name = fmt.Sprintf("New%s%s", r, status)
	}
	code, pointer, clientArgs := sds.buildResponseResultInitCode(resp, result, resAttr, origin, httpclictx, svcctx, headersData, cookiesData, e, sd)
	return &InitData{
		Name:                     name,
		Description:              fmt.Sprintf("%s builds a %q service %q endpoint result from a HTTP %q response.", name, sd.Service.Name, e.Name(), status),
		ClientArgs:               clientArgs,
		ReturnTypeName:           tname,
		ReturnTypeRef:            tref,
		ReturnIsStruct:           expr.IsObject(result.Type),
		ReturnTypeAttribute:      codegen.Goify(origin, true),
		ReturnTypePkg:            pkg,
		ReturnIsPrimitivePointer: pointer,
		ClientCode:               code,
	}
}

func (sds *ServicesData) buildResponseResultInitCode(
	resp *expr.HTTPResponseExpr,
	result *expr.AttributeExpr,
	resAttr *expr.AttributeExpr,
	origin string,
	httpclictx *codegen.AttributeContext,
	svcctx *codegen.AttributeContext,
	headersData []*HeaderData,
	cookiesData []*CookieData,
	e *expr.HTTPEndpointExpr,
	sd *ServiceData,
) (string, bool, []*InitArgData) {
	clientArgs := make([]*InitArgData, 0, 1+len(headersData)+len(cookiesData))
	code := ""
	pointer := false
	var err error
	if resp.Body.Type != expr.Empty {
		if origin != "" {
			pointer = result.IsPrimitivePointer(origin, true)
		}
		if expr.IsObject(resp.Body.Type) {
			pointer = false
		}
		var vcode string
		if ut, ok := resp.Body.Type.(expr.UserType); ok {
			if val := ut.Attribute().Validation; val != nil {
				vcode = codegen.ValidationCode(ut.Attribute(), ut, httpclictx, true, expr.IsAlias(ut), false, "body")
			}
		}
		bodyArg := buildBodyInitArg(sd.Scope, resp.Body, true)
		bodyArg.AttributeData.Validate = vcode
		clientArgs = append(clientArgs, bodyArg)
		var helpers []*codegen.TransformFunctionData
		code, helpers, err = unmarshal(resp.Body, resAttr, "body", httpclictx, svcctx)
		if err == nil {
			sd.ClientTransformHelpers = codegen.AppendHelpers(sd.ClientTransformHelpers, helpers)
		}
	} else if expr.IsArray(result.Type) || expr.IsMap(result.Type) {
		if params := expr.AsObject(e.QueryParams().Type); len(*params) > 0 {
			var helpers []*codegen.TransformFunctionData
			code, helpers, err = unmarshal((*params)[0].Attribute, result, codegen.Goify((*params)[0].Name, false), httpclictx, svcctx)
			if err == nil {
				sd.ClientTransformHelpers = codegen.AppendHelpers(sd.ClientTransformHelpers, helpers)
			}
		}
	}
	if err != nil {
		fmt.Println(err.Error()) // TBD validate DSL so errors are not possible
	}
	clientArgs = append(clientArgs, buildHeaderInitArgs(headersData)...)
	clientArgs = append(clientArgs, buildCookieInitArgs(cookiesData)...)
	return code, pointer, clientArgs
}

func responseTagData(resp *expr.HTTPResponseExpr, result *expr.AttributeExpr, viewed bool) (string, string, bool) {
	if resp.Tag[0] == "" {
		return "", "", false
	}
	return codegen.Goify(resp.Tag[0], true), resp.Tag[1], viewed || result.IsPrimitivePointer(resp.Tag[0], true)
}

// buildErrorsData builds the error data for all the error responses in the
// endpoint expression. The response headers, cookies and body for each response
// are inferred from the method's error expression if not specified explicitly.
func (sds *ServicesData) buildErrorsData(e *expr.HTTPEndpointExpr, sd *ServiceData) []*ErrorGroupData {
	var (
		svc        = sd.Service
		ep         = svc.Method(e.MethodExpr.Name)
		httpclictx = httpContext(sd.Scope, false, false)
	)

	data := make(map[string][]*ErrorData)
	for _, httpError := range e.HTTPErrors {
		ref, errorData := sds.buildSingleErrorData(e, httpError, ep, svc, httpclictx, sd)
		data[ref] = append(data[ref], errorData)
	}
	keys := make([]string, len(data))
	i := 0
	for k := range data {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	var vals []*ErrorGroupData
	for _, k := range keys {
		es := data[k]
		for _, e := range es {
			found := false
			for _, eg := range vals {
				if eg.StatusCode == e.Response.StatusCode {
					eg.Errors = append(eg.Errors, e)
					found = true
					break
				}
			}
			if !found {
				vals = append(vals,
					&ErrorGroupData{
						StatusCode: e.Response.StatusCode,
						Errors:     []*ErrorData{e},
					})
			}
		}
	}
	return vals
}

func (sds *ServicesData) buildSingleErrorData(
	e *expr.HTTPEndpointExpr,
	httpError *expr.HTTPErrorExpr,
	ep *service.MethodData,
	svc *service.Data,
	httpclictx *codegen.AttributeContext,
	sd *ServiceData,
) (string, *ErrorData) {
	httpError.Response.Body = makeHTTPType(httpError.Response.Body)
	pkg := pkgWithDefault(ep.ErrorLocs[httpError.Name], svc.PkgName)
	errctx := serviceContext(pkg, sd.Service.Scope)
	init := sds.buildErrorResultInit(e, httpError, ep, pkg, httpclictx, errctx, svc, sd)
	responseData := sds.buildErrorResponseData(e, httpError, ep, errctx, init, svc, sd)
	ref := svc.Scope.GoFullTypeRef(httpError.AttributeExpr, pkg)
	return ref, &ErrorData{Name: httpError.Name, Response: responseData, Ref: ref}
}

func (sds *ServicesData) buildErrorResultInit(
	e *expr.HTTPEndpointExpr,
	httpError *expr.HTTPErrorExpr,
	ep *service.MethodData,
	pkg string,
	httpclictx *codegen.AttributeContext,
	errctx *codegen.AttributeContext,
	svc *service.Data,
	sd *ServiceData,
) *InitData {
	body := httpError.Response.Body.Type
	if !needInit(httpError.Type) {
		return nil
	}
	headers := sds.extractHeaders(httpError.Response.Headers, httpError.AttributeExpr, errctx, sd.Scope)
	cookies := sds.extractResponseCookies(httpError.Response.Cookies, httpError.AttributeExpr, errctx, sd.Scope)
	args := make([]*InitArgData, 0, len(headers)+len(cookies)+1)
	if body != expr.Empty {
		args = append(args, buildBodyInitArg(sd.Scope, httpError.Response.Body, true))
	}
	args = append(args, buildHeaderInitArgs(headers)...)
	args = append(args, buildCookieInitArgs(cookies)...)
	code, origin := sds.buildErrorResultInitCode(e, httpError, httpclictx, errctx, sd)
	name := fmt.Sprintf("New%s%s", codegen.Goify(ep.Name, true), codegen.Goify(httpError.ErrorExpr.Name, true))
	return &InitData{
		Name:                name,
		Description:         fmt.Sprintf("%s builds a %s service %s endpoint %s error.", name, svc.Name, e.Name(), httpError.ErrorExpr.Name),
		ClientArgs:          args,
		ReturnTypeName:      svc.Scope.GoFullTypeName(httpError.AttributeExpr, pkg),
		ReturnTypeRef:       svc.Scope.GoFullTypeRef(httpError.AttributeExpr, pkg),
		ReturnIsStruct:      expr.IsObject(httpError.Type),
		ReturnTypeAttribute: codegen.Goify(origin, true),
		ReturnTypePkg:       pkg,
		ClientCode:          code,
	}
}

func (sds *ServicesData) buildErrorResultInitCode(
	e *expr.HTTPEndpointExpr,
	httpError *expr.HTTPErrorExpr,
	httpclictx *codegen.AttributeContext,
	errctx *codegen.AttributeContext,
	sd *ServiceData,
) (string, string) {
	body := httpError.Response.Body.Type
	origin := ""
	code := ""
	var err error
	if body != expr.Empty {
		errAtt := httpError.AttributeExpr
		if o, ok := httpError.Response.Body.Meta["origin:attribute"]; ok {
			origin = o[0]
			errAtt = expr.AsObject(httpError.ErrorExpr.Type).Attribute(origin)
		}
		var helpers []*codegen.TransformFunctionData
		code, helpers, err = unmarshal(httpError.Response.Body, errAtt, "body", httpclictx, errctx)
		if err == nil {
			sd.ClientTransformHelpers = codegen.AppendHelpers(sd.ClientTransformHelpers, helpers)
		}
	} else if expr.IsArray(httpError.Type) || expr.IsMap(httpError.Type) {
		if params := expr.AsObject(e.QueryParams().Type); len(*params) > 0 {
			var helpers []*codegen.TransformFunctionData
			code, helpers, err = unmarshal((*params)[0].Attribute, httpError.AttributeExpr, codegen.Goify((*params)[0].Name, false), httpclictx, errctx)
			if err == nil {
				sd.ClientTransformHelpers = codegen.AppendHelpers(sd.ClientTransformHelpers, helpers)
			}
		}
	}
	if err != nil {
		fmt.Println(err.Error()) // TBD validate DSL so errors are not possible
	}
	return code, origin
}

func (sds *ServicesData) buildErrorResponseData(
	e *expr.HTTPEndpointExpr,
	httpError *expr.HTTPErrorExpr,
	ep *service.MethodData,
	errctx *codegen.AttributeContext,
	init *InitData,
	svc *service.Data,
	sd *ServiceData,
) *ResponseData {
	serverBodyData, clientBodyData := sds.buildErrorResponseBodyData(e, httpError, ep, svc, sd)
	headers := sds.extractHeaders(httpError.Response.Headers, httpError.AttributeExpr, errctx, sd.Scope)
	cookies := sds.extractResponseCookies(httpError.Response.Cookies, httpError.AttributeExpr, errctx, sd.Scope)
	contentType := ""
	if httpError.Response.ContentType != expr.ErrorResultIdentifier {
		contentType = httpError.Response.ContentType
	}
	return &ResponseData{
		StatusCode:   statusCodeToHTTPConst(httpError.Response.StatusCode),
		Code:         httpError.Response.StatusCode,
		Headers:      headers,
		ContentType:  contentType,
		Cookies:      cookies,
		ErrorHeader:  httpError.Name,
		ServerBody:   serverBodyData,
		ClientBody:   clientBodyData,
		ResultInit:   init,
		MustValidate: responseFieldsNeedValidation(headers, cookies),
	}
}

func (sds *ServicesData) buildErrorResponseBodyData(
	e *expr.HTTPEndpointExpr,
	httpError *expr.HTTPErrorExpr,
	ep *service.MethodData,
	svc *service.Data,
	sd *ServiceData,
) ([]*TypeData, *TypeData) {
	var serverBodyData []*TypeData
	errorLoc := ep.ErrorLocs[httpError.ErrorExpr.Name]
	if sbd := sds.buildResponseBodyType(httpError.Response.Body, httpError.AttributeExpr, errorLoc, e, true, nil, sd); sbd != nil {
		serverBodyData = append(serverBodyData, sbd)
	}
	clientBodyData := sds.buildResponseBodyType(httpError.Response.Body, httpError.AttributeExpr, errorLoc, e, false, nil, sd)
	if clientBodyData != nil {
		if clientBodyData.Def != "" {
			sd.ClientTypeNames[clientBodyData.Name] = false
		}
		clientBodyData.Description = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP response body for the %q error.",
			clientBodyData.VarName, svc.Name, e.Name(), httpError.Name)
		serverBodyData[0].Description = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP response body for the %q error.",
			serverBodyData[0].VarName, svc.Name, e.Name(), httpError.Name)
	}
	return serverBodyData, clientBodyData
}

// buildRequestBodyType builds the TypeData for a request body. The data makes
// it possible to generate a function on the client side that creates the body
// from the service method payload.
//
// body is the HTTP request body
//
// att is the payload attribute
//
// e is the HTTP endpoint expression
//
// svr is true if the function is generated for server side code.
//
// sd is the service data
func (sds *ServicesData) buildRequestBodyType(body, att *expr.AttributeExpr, e *expr.HTTPEndpointExpr, svr bool, sd *ServiceData) *TypeData {
	if body.Type == expr.Empty {
		return nil
	}
	var (
		name               string
		varname            string
		desc               string
		def                string
		ref                string
		validateDef        string
		validateRef        string
		flatFormUnionField string

		svc     = sd.Service
		httpctx = httpContext(sd.Scope, true, svr)
		ep      = sd.Service.Method(e.Name())
		pkg     = pkgWithDefault(ep.PayloadLoc, sd.Service.PkgName)
		svcctx  = serviceContext(pkg, sd.Service.Scope)
	)
	name = body.Type.Name()
	ref = sd.Scope.GoTypeRef(body)

	addMarshalTags(body, make(map[string]struct{}))

	if ut, ok := body.Type.(expr.UserType); ok {
		varname = codegen.Goify(ut.Name(), true)
		def = goTypeDef(sd.Scope, ut.Attribute(), svr, !svr)
		desc = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP request body.",
			varname, svc.Name, e.Name())
		if e.FormRequest {
			if obj := expr.AsObject(ut.Attribute().Type); obj != nil && len(*obj) == 1 && expr.IsUnion((*obj)[0].Attribute.Type) {
				flatFormUnionField = codegen.Goify((*obj)[0].Name, true)
			}
		}
		// Generate validation code for unmarshaled request bodies on the server,
		// and for client request bodies only when constructor unions require the
		// corresponding validator helper during CLI payload validation.
		if svr || containsUnionType(body.Type) {
			validateDef = codegen.ValidationCode(body, ut, httpctx, true, expr.IsAlias(body.Type), false, "body")
			if validateDef != "" {
				validateRef = fmt.Sprintf("err = Validate%s(&body)", varname)
			}
		}
	} else {
		// Generate validation code first because inline struct validation is removed.
		ctx := codegen.NewAttributeContext(!expr.IsPrimitive(body.Type), false, !svr, "", sd.Scope)
		validateRef = codegen.ValidationCode(body, nil, ctx, true, expr.IsAlias(body.Type), false, "body")
		if svr && expr.IsObject(body.Type) {
			// Body is an explicit object described in the design and in
			// this case the GoTypeRef is an inline struct definition. We
			// want to force all attributes to be pointers because we are
			// generating the server body type pre-validation.
			body.Validation = nil
		}
		varname = sd.Scope.GoTypeRef(body)
		desc = body.Description
	}
	var init *InitData
	if !svr && att.Type != expr.Empty && needInit(body.Type) {
		var (
			name    string
			desc    string
			code    string
			origin  string
			err     error
			helpers []*codegen.TransformFunctionData

			sourceVar = "p"
			svc       = sd.Service
		)
		{
			name = fmt.Sprintf("New%s", codegen.Goify(sd.Scope.GoTypeName(body), true))
			desc = fmt.Sprintf("%s builds the HTTP request body from the payload of the %q endpoint of the %q service.",
				name, e.Name(), svc.Name)
			src := sourceVar
			srcAtt := att
			// If design uses Body("name") syntax then need to use payload attribute
			// to transform.
			if o, ok := body.Meta["origin:attribute"]; ok {
				srcObj := expr.AsObject(att.Type)
				origin = o[0]
				srcAtt = srcObj.Attribute(origin)
				src += "." + codegen.Goify(origin, true)
			}
			code, helpers, err = marshal(srcAtt, body, src, "body", svcctx, httpctx)
			if err != nil {
				fmt.Println(err.Error()) // TBD validate DSL so errors are not possible
			}
			sd.ClientTransformHelpers = codegen.AppendHelpers(sd.ClientTransformHelpers, helpers)
		}
		arg := InitArgData{
			Ref: sourceVar,
			AttributeData: &AttributeData{
				Name:     "payload",
				VarName:  sourceVar,
				TypeRef:  svc.Scope.GoFullTypeRef(att, pkg),
				Type:     att.Type,
				Validate: validateDef,
				Example:  att.Example(sds.Root.API.ExampleGenerator),
			},
		}
		init = &InitData{
			Name:                name,
			Description:         desc,
			ReturnTypeRef:       sd.Scope.GoTypeRef(body),
			ReturnTypeAttribute: codegen.Goify(origin, true),
			ClientCode:          code,
			ClientArgs:          []*InitArgData{&arg},
		}
	}
	return &TypeData{
		Name:               name,
		VarName:            varname,
		Description:        desc,
		Def:                def,
		Ref:                ref,
		Init:               init,
		ValidateDef:        validateDef,
		ValidateRef:        validateRef,
		Example:            body.Example(sds.Root.API.ExampleGenerator),
		FlatFormUnionField: flatFormUnionField,
	}
}

// buildResponseBodyType builds the TypeData for a response body. The data
// makes it possible to generate a function that creates the server response
// body from the service method result/projected result or error.
//
// body is the response (success or error) HTTP body.
//
// att is the result/projected attribute.
//
// svr is true if the function is generated for server side code
//
// view is the view name to add as a suffix to the type name.
func (sds *ServicesData) buildResponseBodyType(body, att *expr.AttributeExpr, loc *codegen.Location, e *expr.HTTPEndpointExpr, svr bool, view *string, sd *ServiceData) *TypeData {
	if body.Type == expr.Empty {
		return nil
	}
	var (
		name        string
		varname     string
		desc        string
		def         string
		ref         string
		validateDef string
		validateRef string
		viewName    string
		mustInit    bool

		svc     = sd.Service
		httpctx = httpContext(sd.Scope, false, svr)
		pkg     = pkgWithDefault(loc, sd.Service.PkgName)
		svcctx  = serviceContext(pkg, sd.Service.Scope)
	)
	// For server code, we project the response body type if the type is a result
	// type and generate a type for each view in the result type. This makes it
	// possible to return only the attributes in the view in the server response.
	if svr && view != nil && *view != "" {
		viewName = *view
		body = expr.DupAtt(body)
		if rt, ok := body.Type.(*expr.ResultTypeExpr); ok {
			var err error
			rt, err = expr.Project(rt, *view)
			if err != nil {
				panic(err)
			}
			body.Type = rt
			sd.ServerTypeNames[rt.Name()] = false
		}
	}

	name = body.Type.Name()
	ref = sd.Scope.GoTypeRef(body)
	mustInit = att.Type != expr.Empty && needInit(body.Type)

	addMarshalTags(body, make(map[string]struct{}))

	if ut, ok := body.Type.(expr.UserType); ok {
		// response body is a user type.
		varname = codegen.Goify(ut.Name(), true)
		def = goTypeDef(sd.Scope, ut.Attribute(), !svr, svr)
		desc = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP response body.",
			varname, svc.Name, e.Name())
		if !svr && view == nil {
			// generate validation code for unmarshaled type (client-side).
			validateDef = codegen.ValidationCode(body, ut, httpctx, true, expr.IsAlias(body.Type), false, "body")
			if validateDef != "" {
				target := "&body"
				if expr.IsArray(ut) {
					// result type collection
					target = "body"
				}
				validateRef = fmt.Sprintf("err = Validate%s(%s)", varname, target)
			}
		}
	} else if !expr.IsPrimitive(body.Type) && mustInit {
		// Response body is an array or map type.
		//
		// Server-side code needs a named wrapper (scoped to the endpoint) so the
		// generator can produce stable constructor identifiers (e.g.
		// New<Endpoint>ResponseBody) for element-wise transforms and projections.
		//
		// Client-side code decodes directly into the concrete composite type (e.g.
		// []T, map[K]V) and validates/transforms the value in-place. This avoids
		// generating endpoint-named alias types that are structurally identical and
		// may be deduplicated away in client/types.go.
		if svr {
			name = codegen.Goify(e.Name(), true) + "ResponseBody"
			varname = name
			desc = fmt.Sprintf("%s is the type of the %q service %q endpoint HTTP response body.",
				varname, svc.Name, e.Name())
			def = goTypeDef(sd.Scope, body, !svr, svr)
		} else {
			varname = sd.Scope.GoTypeRef(body)
			desc = body.Description
			def = ""
		}
		validateRef = codegen.ValidationCode(body, nil, httpctx, true, expr.IsAlias(body.Type), false, "body")
	} else {
		// response body is a primitive type. They are used as non-pointers when
		// encoding/decoding responses.
		httpctx = httpContext(sd.Scope, false, true)
		validateRef = codegen.ValidationCode(body, nil, httpctx, true, expr.IsAlias(body.Type), false, "body")
		varname = sd.Scope.GoTypeRef(body)
		desc = body.Description
	}
	if svr {
		sd.ServerTypeNames[name] = false
		// We collect the server body types need to generate a response body type
		// here because the response body type would be different from the actual
		// type in the HTTPResponseExpr since we projected the body type above.
		// For client side, we don't have to generate a separate body type per
		// view. Hence the client types are collected in "analyze" function.
		collectUserTypes(body.Type, func(ut expr.UserType) {
			if d := sds.attributeTypeData(ut, false, false, true, sd); d != nil {
				sd.ServerBodyAttributeTypes = append(sd.ServerBodyAttributeTypes, d)
			}
		})
	}

	var init *InitData
	if svr && mustInit {
		var (
			name    string
			desc    string
			rtref   string
			code    string
			origin  string
			err     error
			helpers []*codegen.TransformFunctionData

			sourceVar = "res"
			svc       = sd.Service
		)
		{
			var rtname string
			if _, ok := body.Type.(expr.UserType); !ok && !expr.IsPrimitive(body.Type) {
				rtname = codegen.Goify(e.Name(), true) + "ResponseBody"
				rtref = rtname
			} else {
				rtname = codegen.Goify(sd.Scope.GoTypeName(body), true)
				rtref = sd.Scope.GoTypeRef(body)
			}
			name = fmt.Sprintf("New%s", rtname)
			desc = fmt.Sprintf("%s builds the HTTP response body from the result of the %q endpoint of the %q service.",
				name, e.Name(), svc.Name)
			if view != nil {
				svcctx = viewContext(sd.Service.ViewsPkg, sd.Service.ViewScope)
			}
			src := sourceVar
			srcAtt := att
			// If design uses Body("name") syntax then need to use result attribute
			// to transform.
			if o, ok := body.Meta["origin:attribute"]; ok {
				srcObj := expr.AsObject(att.Type)
				origin = o[0]
				srcAtt = srcObj.Attribute(origin)
				src += "." + codegen.Goify(origin, true)
			}
			code, helpers, err = marshal(srcAtt, body, src, "body", svcctx, httpctx)
			if err != nil {
				panic(err) // bug
			}
			sd.ServerTransformHelpers = codegen.AppendHelpers(sd.ServerTransformHelpers, helpers)
		}
		ref := sourceVar
		if view != nil {
			ref += ".Projected"
		}
		tref := svc.Scope.GoFullTypeRef(att, pkg)
		if view != nil {
			tref = svc.ViewScope.GoFullTypeRef(att, svc.ViewsPkg)
		}
		arg := InitArgData{
			Ref: ref,
			AttributeData: &AttributeData{
				Name:     "result",
				VarName:  sourceVar,
				TypeRef:  tref,
				Type:     att.Type,
				Validate: validateDef,
				Example:  att.Example(sds.Root.API.ExampleGenerator),
			},
		}
		init = &InitData{
			Name:                name,
			Description:         desc,
			ReturnTypeRef:       rtref,
			ReturnTypeAttribute: codegen.Goify(origin, true),
			ServerCode:          code,
			ServerArgs:          []*InitArgData{&arg},
		}
	}
	td := &TypeData{
		Name:        name,
		VarName:     varname,
		Description: desc,
		Def:         def,
		Ref:         ref,
		Init:        init,
		ValidateDef: validateDef,
		ValidateRef: validateRef,
		Example:     body.Example(sds.Root.API.ExampleGenerator),
		View:        viewName,
	}
	return td
}

func (sds *ServicesData) extractPathParams(a *expr.MappedAttributeExpr, service *expr.AttributeExpr, scope *codegen.NameScope) []*ParamData {
	var params []*ParamData
	codegen.WalkMappedAttr(a, func(name, elem string, _ bool, c *expr.AttributeExpr) error { // nolint: errcheck
		// The StringSlice field of ParamData must be false for aliased primitive types
		var stringSlice bool
		if arr := expr.AsArray(c.Type); arr != nil {
			stringSlice = arr.ElemType.Type.Kind() == expr.StringKind
		}

		c = makeHTTPType(c)
		var (
			varn = scope.Name(codegen.Goify(name, false))
			arr  = expr.AsArray(c.Type)
			ctx  = serviceContext("", scope)
			ft   = service.Type

			fptr bool
		)
		fieldName := codegen.GoifyAtt(c, name, true)
		if !expr.IsObject(service.Type) {
			fieldName = ""
		} else {
			fptr = service.IsPrimitivePointer(name, true)
			ft = service.Find(name).Type
		}
		params = append(params, &ParamData{
			Map:            false,
			MapStringSlice: false,
			Element: &Element{
				HTTPName:      elem,
				AttributeName: name,
				Slice:         arr != nil,
				StringSlice:   stringSlice,
				AttributeData: &AttributeData{
					Name:         name,
					Description:  c.Description,
					FieldName:    fieldName,
					FieldPointer: fptr,
					FieldType:    ft,
					VarName:      varn,
					Required:     true,
					Type:         c.Type,
					TypeName:     scope.GoTypeName(c),
					TypeRef:      scope.GoTypeRef(c),
					Pointer:      false,
					Validate:     codegen.AttributeValidationCode(c, nil, ctx, true, expr.IsAlias(c.Type), varn, name),
					DefaultValue: c.DefaultValue,
					Example:      c.Example(sds.Root.API.ExampleGenerator),
				},
			},
		})
		return nil
	})

	return params
}

func (sds *ServicesData) extractQueryParams(a *expr.MappedAttributeExpr, service *expr.AttributeExpr, scope *codegen.NameScope) []*ParamData {
	var params []*ParamData
	codegen.WalkMappedAttr(a, func(name, elem string, required bool, c *expr.AttributeExpr) error { // nolint: errcheck
		// The StringSlice field of ParamData must be false for aliased primitive types
		var stringSlice bool
		if arr := expr.AsArray(c.Type); arr != nil {
			stringSlice = arr.ElemType.Type.Kind() == expr.StringKind
		}

		c = makeHTTPType(c)
		var (
			varn    = scope.Name(codegen.Goify(name, false))
			arr     = expr.AsArray(c.Type)
			mp      = expr.AsMap(c.Type)
			typeRef = scope.GoTypeRef(c)
			ctx     = serviceContext("", scope)
			ft      = service.Type

			pointer bool
			fptr    bool
		)
		pointer = a.IsPrimitivePointer(name, true)
		if pointer {
			typeRef = "*" + typeRef
		}
		fieldName := codegen.GoifyAtt(c, name, true)
		if !expr.IsObject(service.Type) {
			fieldName = ""
		} else {
			fptr = service.IsPrimitivePointer(name, true)
			ft = service.Find(name).Type
		}
		params = append(params, &ParamData{
			Map: mp != nil,
			MapStringSlice: mp != nil &&
				mp.KeyType.Type.Kind() == expr.StringKind &&
				mp.ElemType.Type.Kind() == expr.ArrayKind &&
				expr.AsArray(mp.ElemType.Type).ElemType.Type.Kind() == expr.StringKind,
			Element: &Element{
				Slice:         arr != nil,
				StringSlice:   stringSlice,
				HTTPName:      elem,
				AttributeName: name,
				AttributeData: &AttributeData{
					Name:         name,
					Description:  c.Description,
					FieldName:    fieldName,
					FieldPointer: fptr,
					FieldType:    ft,
					VarName:      varn,
					Required:     required,
					Type:         c.Type,
					TypeName:     scope.GoTypeName(c),
					TypeRef:      typeRef,
					Pointer:      pointer,
					Validate:     codegen.AttributeValidationCode(c, nil, ctx, required, expr.IsAlias(c.Type), varn, name),
					DefaultValue: c.DefaultValue,
					Example:      c.Example(sds.Root.API.ExampleGenerator),
				},
			},
		})
		return nil
	})

	return params
}

func (sds *ServicesData) extractHeaders(a *expr.MappedAttributeExpr, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext, scope *codegen.NameScope) []*HeaderData {
	var headers []*HeaderData
	codegen.WalkMappedAttr(a, func(name, elem string, required bool, _ *expr.AttributeExpr) error { // nolint: errcheck
		var attr *expr.AttributeExpr
		if attr = svcAtt.Find(name); attr == nil {
			attr = svcAtt
		}
		var hattr *expr.AttributeExpr
		var stringSlice bool
		// The StringSlice field of ParamData must be false for aliased primitive types
		if arr := expr.AsArray(attr.Type); arr != nil {
			stringSlice = arr.ElemType.Type.Kind() == expr.StringKind
		}

		hattr = makeHTTPType(attr)
		var (
			varn    = scope.Name(codegen.Goify(name, false))
			arr     = expr.AsArray(hattr.Type)
			typeRef = scope.GoTypeRef(hattr)
			ft      = attr.Type

			fieldName string
			pointer   bool
			fptr      bool
		)
		pointer = a.IsPrimitivePointer(name, true)
		if expr.IsObject(svcAtt.Type) {
			fieldName = codegen.GoifyAtt(attr, name, true)
			fptr = svcCtx.IsPrimitivePointer(name, svcAtt)
		}
		if pointer {
			typeRef = "*" + typeRef
		}
		headers = append(headers, &HeaderData{
			CanonicalName: http.CanonicalHeaderKey(elem),
			Element: &Element{
				HTTPName:      elem,
				Slice:         arr != nil,
				StringSlice:   stringSlice,
				AttributeName: name,
				AttributeData: &AttributeData{
					Name:         name,
					Description:  hattr.Description,
					FieldName:    fieldName,
					FieldPointer: fptr,
					FieldType:    ft,
					VarName:      varn,
					TypeName:     scope.GoTypeName(hattr),
					TypeRef:      typeRef,
					Required:     required,
					Pointer:      pointer,
					Type:         hattr.Type,
					Validate:     codegen.AttributeValidationCode(hattr, nil, svcCtx, required, expr.IsAlias(hattr.Type), varn, name),
					DefaultValue: hattr.DefaultValue,
					Example:      hattr.Example(sds.Root.API.ExampleGenerator),
				},
			},
		})
		return nil
	})
	return headers
}

func (sds *ServicesData) extractCookies(a *expr.MappedAttributeExpr, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext, scope *codegen.NameScope) []*CookieData {
	var cookies []*CookieData
	codegen.WalkMappedAttr(a, func(name, elem string, required bool, _ *expr.AttributeExpr) error { // nolint: errcheck
		pointer := a.IsPrimitivePointer(name, true)
		cookies = append(cookies, sds.cookieData(name, elem, required, pointer, svcAtt, svcCtx, scope))
		return nil
	})
	return cookies
}

func (sds *ServicesData) extractResponseCookies(cookiesExpr []*expr.HTTPResponseCookieExpr, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext, scope *codegen.NameScope) []*CookieData {
	cookies := make([]*CookieData, 0, len(cookiesExpr))
	for _, cookieExpr := range cookiesExpr {
		name := cookieExpr.AttributeName()
		if name == "" {
			continue
		}
		cookie := sds.cookieData(name, cookieExpr.HTTPName(), cookieExpr.IsRequired(name), cookieExpr.IsPrimitivePointer(name, true), svcAtt, svcCtx, scope)
		cookie.MaxAge = cookieExpr.MaxAge
		cookie.Path = cookieExpr.Path
		cookie.Domain = cookieExpr.Domain
		cookie.Secure = cookieExpr.Secure
		cookie.HTTPOnly = cookieExpr.HTTPOnly
		switch cookieExpr.SameSite {
		case expr.CookieSameSiteLax:
			cookie.SameSite = "http.SameSiteLaxMode"
		case expr.CookieSameSiteStrict:
			cookie.SameSite = "http.SameSiteStrictMode"
		case expr.CookieSameSiteNone:
			cookie.SameSite = "http.SameSiteNoneMode"
		case expr.CookieSameSiteDefault:
			cookie.SameSite = "http.SameSiteDefaultMode"
		}
		cookies = append(cookies, cookie)
	}
	return cookies
}

func (sds *ServicesData) cookieData(name, elem string, required bool, pointer bool, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext, scope *codegen.NameScope) *CookieData {
	var hattr *expr.AttributeExpr
	if hattr = svcAtt.Find(name); hattr == nil {
		hattr = svcAtt
	}
	hattr = makeHTTPType(hattr)
	var (
		varn    = scope.Name(codegen.Goify(name, false))
		typeRef = scope.GoTypeRef(hattr)
		ft      = svcAtt.Type

		fieldName string
		fptr      bool
	)
	if expr.IsObject(svcAtt.Type) {
		fieldName = codegen.GoifyAtt(hattr, name, true)
		fptr = svcCtx.IsPrimitivePointer(name, svcAtt)
		ft = svcAtt.Find(name).Type
	}
	if pointer {
		typeRef = "*" + typeRef
	}
	return &CookieData{
		Element: &Element{
			HTTPName:      elem,
			AttributeName: name,
			AttributeData: &AttributeData{
				Name:         name,
				Description:  hattr.Description,
				FieldName:    fieldName,
				FieldPointer: fptr,
				FieldType:    ft,
				VarName:      varn,
				TypeName:     scope.GoTypeName(hattr),
				TypeRef:      typeRef,
				Required:     required,
				Pointer:      pointer,
				Type:         hattr.Type,
				Validate:     codegen.AttributeValidationCode(hattr, nil, svcCtx, required, expr.IsAlias(hattr.Type), varn, name),
				DefaultValue: hattr.DefaultValue,
				Example:      hattr.Example(sds.Root.API.ExampleGenerator),
			},
		},
	}
}

// collectUserTypes traverses the given data type recursively and calls back the
// given function for each attribute using a user type.
func collectUserTypes(dt expr.DataType, cb func(expr.UserType), seen ...map[string]struct{}) {
	if dt == expr.Empty {
		return
	}
	var s map[string]struct{}
	if len(seen) > 0 {
		s = seen[0]
	} else {
		s = make(map[string]struct{})
	}
	switch actual := dt.(type) {
	case *expr.Object:
		for _, nat := range *actual {
			collectUserTypes(nat.Attribute.Type, cb, seen...)
		}
	case *expr.Union:
		for _, nat := range actual.Values {
			collectUserTypes(nat.Attribute.Type, cb, seen...)
		}
	case *expr.Array:
		collectUserTypes(actual.ElemType.Type, cb, seen...)
	case *expr.Map:
		collectUserTypes(actual.KeyType.Type, cb, seen...)
		collectUserTypes(actual.ElemType.Type, cb, seen...)
	case expr.UserType:
		if _, ok := s[actual.ID()]; ok {
			return
		}
		s[actual.ID()] = struct{}{}
		cb(actual)
		collectUserTypes(actual.Attribute().Type, cb, s)
	}
}

func collectUnionBranchUserTypes(att *expr.AttributeExpr, ids map[string]struct{}) {
	collectUnionBranchUserTypesSeen(att, ids, make(map[string]struct{}))
}

func collectUnionBranchUserTypesSeen(att *expr.AttributeExpr, ids, seen map[string]struct{}) {
	if att == nil || att.Type == expr.Empty {
		return
	}
	switch actual := att.Type.(type) {
	case expr.UserType:
		if _, ok := seen[actual.ID()]; ok {
			return
		}
		seen[actual.ID()] = struct{}{}
		collectUnionBranchUserTypesSeen(actual.Attribute(), ids, seen)
	case *expr.Object:
		for _, nat := range *actual {
			collectUnionBranchUserTypesSeen(nat.Attribute, ids, seen)
		}
	case *expr.Array:
		collectUnionBranchUserTypesSeen(actual.ElemType, ids, seen)
	case *expr.Map:
		collectUnionBranchUserTypesSeen(actual.KeyType, ids, seen)
		collectUnionBranchUserTypesSeen(actual.ElemType, ids, seen)
	case *expr.Union:
		for _, nat := range actual.Values {
			collectUserTypes(nat.Attribute.Type, func(ut expr.UserType) {
				ids[ut.ID()] = struct{}{}
			})
			collectUnionBranchUserTypesSeen(nat.Attribute, ids, seen)
		}
	}
}

func collectHTTPUnionTypes(att *expr.AttributeExpr, scope *codegen.NameScope, unions map[string]*service.UnionTypeData, seen map[string]struct{}) {
	if att == nil || att.Type == expr.Empty {
		return
	}
	switch dt := att.Type.(type) {
	case expr.UserType:
		if _, ok := seen[dt.ID()]; ok {
			return
		}
		seen[dt.ID()] = struct{}{}
		collectHTTPUnionTypes(dt.Attribute(), scope, unions, seen)
	case *expr.Object:
		for _, nat := range sortedNamedAttributes(*dt) {
			collectHTTPUnionTypes(nat.Attribute, scope, unions, seen)
		}
	case *expr.Array:
		collectHTTPUnionTypes(dt.ElemType, scope, unions, seen)
	case *expr.Map:
		collectHTTPUnionTypes(dt.KeyType, scope, unions, seen)
		collectHTTPUnionTypes(dt.ElemType, scope, unions, seen)
	case *expr.Union:
		hash := dt.Hash()
		if _, ok := unions[hash]; !ok {
			unions[hash] = buildHTTPUnionTypeData(dt, scope)
		}
		for _, nat := range dt.Values {
			collectHTTPUnionTypes(nat.Attribute, scope, unions, seen)
		}
	}
}

func buildHTTPUnionTypeData(u *expr.Union, scope *codegen.NameScope) *service.UnionTypeData {
	att := &expr.AttributeExpr{Type: u}
	name := scope.GoTypeName(att)
	kindName := scope.Unique(name + "Kind")

	fields := make([]*service.UnionFieldData, len(u.Values))
	hasScalarFormBranch := false
	for i, nat := range u.Values {
		fieldName := codegen.Goify(nat.Name, true)
		fieldType := scope.GoTypeRef(nat.Attribute)
		kindConst := kindName + fieldName
		fields[i] = &service.UnionFieldData{
			Name:                      nat.Name,
			KindConst:                 kindConst,
			FieldName:                 fieldName,
			FieldType:                 fieldType,
			TypeTag:                   expr.UnionVariantTag(nat),
			FlatFormObject:            expr.IsObject(nat.Attribute.Type),
			FlatFormObjectAllowsEmpty: flatFormObjectAllowsEmpty(nat.Attribute),
			EmptyValueExpr:            emptyObjectValueExpr(fieldType),
			EmitPrimitiveAlias:        false,
		}
		hasScalarFormBranch = hasScalarFormBranch || !fields[i].FlatFormObject
	}

	return &service.UnionTypeData{
		Name:                name,
		KindName:            kindName,
		Fields:              fields,
		TypeKey:             u.GetTypeKey(),
		ValueKey:            u.GetValueKey(),
		HasScalarFormBranch: hasScalarFormBranch,
	}
}

// sortedNamedAttributes returns object fields sorted by attribute name.
// Union naming uses NameScope uniqueness, so callers that discover unions while
// traversing objects must use a deterministic field order to avoid oscillating
// generated identifiers across runs.
func sortedNamedAttributes(attrs []*expr.NamedAttributeExpr) []*expr.NamedAttributeExpr {
	if len(attrs) < 2 {
		return attrs
	}
	sorted := slices.Clone(attrs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}

func flatFormObjectAllowsEmpty(att *expr.AttributeExpr) bool {
	return expr.IsObject(att.Type) && len(att.AllRequired()) == 0
}

func emptyObjectValueExpr(fieldType string) string {
	if strings.HasPrefix(fieldType, "*") {
		return "&" + strings.TrimPrefix(fieldType, "*") + "{}"
	}
	return fieldType + "{}"
}

func (sds *ServicesData) attributeTypeData(ut expr.UserType, req, ptr, server bool, rd *ServiceData) *TypeData {
	if ut == expr.Empty {
		return nil
	}
	seen := rd.ServerTypeNames
	if !server {
		seen = rd.ClientTypeNames
	}
	if _, ok := seen[ut.Name()]; ok {
		return nil
	}
	seen[ut.Name()] = false

	var (
		name        string
		desc        string
		validate    string
		validateRef string

		att  = &expr.AttributeExpr{Type: ut}
		hctx = httpContext(rd.Scope, req, server)
	)
	name = rd.Scope.GoTypeName(att)
	ctx := "request"
	if !req {
		ctx = "response"
	}
	desc = name + " is used to define fields on " + ctx + " body types."
	if (req || !req && !server) && !expr.IsAlias(ut) {
		// Generate validations for responses client-side and for
		// requests server-side and CLI.
		// Alias types are validated inline in the parent type
		validate = codegen.ValidationCode(ut.Attribute(), ut, hctx, true, expr.IsAlias(ut), false, "body")
		if validate == "" && req && !server && needsClientRequestBodyValidatorStub(ut) {
			validate = "// no validations"
		}
	}
	if validate != "" {
		validateRef = fmt.Sprintf("err = Validate%s(v)", name)
	}
	return &TypeData{
		Name:        ut.Name(),
		VarName:     name,
		Description: desc,
		Def:         goTypeDef(rd.Scope, ut.Attribute(), ptr, hctx.UseDefault),
		Ref:         rd.Scope.GoTypeRef(att),
		ValidateDef: validate,
		ValidateRef: validateRef,
		Example:     att.Example(sds.Root.API.ExampleGenerator),
	}
}

func needsClientRequestBodyValidatorStub(ut expr.UserType) bool {
	if ut == nil || ut.Attribute() == nil || ut.Attribute().Meta == nil {
		return false
	}
	_, ok := ut.Attribute().Meta.Last("oneof:type:tag")
	return ok
}

// httpContext returns a context for attributes of types used to marshal and
// unmarshal HTTP requests and responses.
//
// pkg is the package name where the body type exists
//
// scope is the named scope
//
// request if true indicates that the type is a request type, else response
// type
//
// svr if true indicates that the type is a server type, else client type
func httpContext(scope *codegen.NameScope, request, svr bool) *codegen.AttributeContext {
	marshal := !request && svr || request && !svr
	return codegen.NewAttributeContext(!marshal, false, marshal, "", scope)
}

// serviceContext returns an attribute context for service types.
func serviceContext(pkg string, scope *codegen.NameScope) *codegen.AttributeContext {
	return codegen.NewAttributeContext(false, false, true, pkg, scope)
}

// viewContext returns an attribute context for projected types.
func viewContext(pkg string, scope *codegen.NameScope) *codegen.AttributeContext {
	return codegen.NewAttributeContext(true, false, true, pkg, scope)
}

// pkgWithDefault returns the package name of the given location if not nil, def otherwise.
func pkgWithDefault(loc *codegen.Location, def string) string {
	if loc == nil {
		return def
	}
	return loc.PackageName()
}

// unmarshal initializes a data structure defined by target type from a data
// structure defined by source type. The attributes in the source data
// structure are pointers and the attributes in the target data structure that
// have default values are non-pointers. Fields in target type are initialized
// with their default values (if any).
//
// source, target are the attributes used in the transformation
//
// sourceVar, targetVar are the variable names for source and target used in
// the transformation code
//
// sourceCtx, targetCtx are the source and target attribute contexts
func unmarshal(source, target *expr.AttributeExpr, sourceVar string, sourceCtx, targetCtx *codegen.AttributeContext) (string, []*codegen.TransformFunctionData, error) {
	return codegen.GoTransform(source, target, sourceVar, "v", sourceCtx, targetCtx, "unmarshal", true)
}

// marshal initializes a data structure defined by target type from a data
// structure defined by source type. The fields in the source and target
// data structure use non-pointers for attributes with default values.
//
// source, target are the attributes used in the transformation
//
// sourceVar, targetVar are the variable names for source and target used in
// the transformation code
//
// sourceCtx, targetCtx are the source and target attribute contexts
func marshal(source, target *expr.AttributeExpr, sourceVar, targetVar string, sourceCtx, targetCtx *codegen.AttributeContext) (string, []*codegen.TransformFunctionData, error) {
	return codegen.GoTransform(source, target, sourceVar, targetVar, sourceCtx, targetCtx, "marshal", true)
}

// needConversion returns true if the type needs to be converted from a string.
func needConversion(dt expr.DataType) bool {
	if dt == expr.Empty {
		return false
	}
	switch actual := dt.(type) {
	case expr.Primitive:
		if actual.Kind() == expr.StringKind ||
			actual.Kind() == expr.AnyKind ||
			actual.Kind() == expr.BytesKind {
			return false
		}
		return true
	case *expr.Array:
		return needConversion(actual.ElemType.Type)
	case *expr.Map:
		return needConversion(actual.KeyType.Type) ||
			needConversion(actual.ElemType.Type)
	default:
		return true
	}
}

// addMarshalTags adds JSON, XML and Form tags to all inline object attributes recursively.
func addMarshalTags(att *expr.AttributeExpr, seen map[string]struct{}) {
	if ut, ok := att.Type.(expr.UserType); ok {
		if _, ok := seen[ut.Hash()]; ok {
			return // avoid infinite recursions
		}
		seen[ut.Hash()] = struct{}{}
		if expr.IsObject(ut.Attribute().Type) {
			for _, att := range *(expr.AsObject(att.Type)) {
				addMarshalTags(att.Attribute, seen)
			}
		}
		return
	}
	if expr.IsArray(att.Type) {
		addMarshalTags(expr.AsArray(att.Type).ElemType, seen)
		return
	}
	if expr.IsMap(att.Type) {
		addMarshalTags(expr.AsMap(att.Type).KeyType, seen)
		addMarshalTags(expr.AsMap(att.Type).ElemType, seen)
		return
	}
	if !expr.IsObject(att.Type) {
		return
	}
	// inline object
	for _, natt := range *(expr.AsObject(att.Type)) {
		if natt.Attribute.Meta == nil {
			natt.Attribute.Meta = expr.MetaExpr{}
		}
		ns := []string{natt.Name}
		natt.Attribute.Meta["struct:tag:form"] = ns
		natt.Attribute.Meta["struct:tag:json"] = ns
		natt.Attribute.Meta["struct:tag:xml"] = ns
	}
}

func containsUnionType(dt expr.DataType) bool {
	return containsUnionTypeRecursive(dt, make(map[string]struct{}))
}

func containsUnionTypeRecursive(dt expr.DataType, seen map[string]struct{}) bool {
	switch actual := dt.(type) {
	case nil:
		return false
	case *expr.Union:
		return true
	case expr.UserType:
		if _, ok := seen[actual.ID()]; ok {
			return false
		}
		seen[actual.ID()] = struct{}{}
		return containsUnionTypeRecursive(actual.Attribute().Type, seen)
	case *expr.Object:
		for _, nat := range *actual {
			if containsUnionTypeRecursive(nat.Attribute.Type, seen) {
				return true
			}
		}
	case *expr.Array:
		return containsUnionTypeRecursive(actual.ElemType.Type, seen)
	case *expr.Map:
		return containsUnionTypeRecursive(actual.KeyType.Type, seen) || containsUnionTypeRecursive(actual.ElemType.Type, seen)
	}
	return false
}

// needInit returns true if and only if the given type is or makes use of user
// types.
func needInit(dt expr.DataType) bool {
	if dt == expr.Empty {
		return false
	}
	switch actual := dt.(type) {
	case expr.Primitive:
		return false
	case *expr.Array:
		return needInit(actual.ElemType.Type)
	case *expr.Map:
		return needInit(actual.KeyType.Type) ||
			needInit(actual.ElemType.Type)
	case *expr.Object:
		for _, nat := range *actual {
			if needInit(nat.Attribute.Type) {
				return true
			}
		}
		return false
	case expr.UserType:
		return true
	default:
		panic(fmt.Sprintf("unknown data type %T", actual)) // bug
	}
}

// upgradeParams returns the data required to render the websocket_upgrade
// template.
func upgradeParams(e *EndpointData, fn string) map[string]any {
	return map[string]any{
		"ViewedResult": e.Method.ViewedResult,
		"Function":     fn,
	}
}

// NeedDialer returns true if at least one method in the defined services
// uses WebSocket for sending payload or result.
func NeedDialer(data []*ServiceData) bool {
	return slices.ContainsFunc(data, HasWebSocket)
}
