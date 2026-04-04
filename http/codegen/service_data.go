package codegen

import (
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
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
		// ErrorHeader contains the value of the response "loom-error"
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
	irService := transportir.BuildService(httpSvc)
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
	for _, httpEndpoint := range irService.Endpoints {
		sd.Endpoints = append(sd.Endpoints, sds.buildEndpointDataFromIR(httpEndpoint, svc, sd, scope))
	}
	for _, endpointIR := range irService.Endpoints {
		sds.collectEndpointBodyAttributeTypes(endpointIR, sd)
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

func (sds *ServicesData) buildEndpointDataFromIR(endpointIR *transportir.Endpoint, svc *service.Data, sd *ServiceData, scope *codegen.NameScope) *EndpointData {
	method := svc.Method(endpointIR.MethodName)
	routes := sds.buildEndpointRoutes(endpointIR, method, svc, sd)
	payload := sds.buildPayloadDataFromIR(endpointIR, sd)
	reqs, hsch, bosch, qsch, basch := sds.buildRequirementSchemes(endpointIR)
	requestInit := sds.buildClientRequestInit(endpointIR, method, svc, routes)

	endpoint := &EndpointData{
		Method:          method,
		ServiceName:     svc.Name,
		ServiceVarName:  svc.VarName,
		ServicePkgName:  svc.PkgName,
		Payload:         payload,
		Result:          sds.buildResultDataFromIR(endpointIR, sd),
		Errors:          sds.buildErrorsDataFromIR(endpointIR, sd),
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
		HasMixedResults: endpointIR.Response.HasMixedResults,
		RequestEncoder:  endpointRequestEncoderName(method, payload, basch),
		ResponseDecoder: fmt.Sprintf("Decode%sResponse", method.VarName),
		Requirements:    reqs,
	}
	if endpointIR.Stream.IsStreaming {
		sds.initWebSocketData(endpoint, endpointIR, sd)
		initSSEData(endpoint, endpointIR, sd)
	}
	sds.initEndpointMultipartData(endpoint, endpointIR, method, svc)
	if endpointIR.Request.SkipBodyEncode {
		endpoint.BuildStreamPayload = scope.Unique("Build" + codegen.Goify(method.Name, true) + "StreamPayload")
	}
	if endpointIR.Redirect != nil {
		endpoint.Redirect = &RedirectData{
			URL:        endpointIR.Redirect.URL,
			StatusCode: statusCodeToHTTPConst(endpointIR.Redirect.StatusCode),
		}
	}
	return endpoint
}

func (sds *ServicesData) buildEndpointRoutes(endpointIR *transportir.Endpoint, method *service.MethodData, svc *service.Data, sd *ServiceData) []*RouteData {
	routes := make([]*RouteData, 0, len(endpointIR.Routes))
	for index, route := range endpointIR.Routes {
		routes = append(routes, &RouteData{
			Verb:     route.Method,
			Path:     route.Path,
			PathInit: sds.buildPathInitData(endpointIR, method, svc, sd, route.Path, index),
		})
	}
	return routes
}

func (sds *ServicesData) buildPathInitData(endpointIR *transportir.Endpoint, method *service.MethodData, svc *service.Data, sd *ServiceData, path string, pathCount int) *InitData {
	params := expr.ExtractHTTPWildcards(path)
	initArgs := make([]*InitArgData, len(params))
	pathParamsObj := pathParametersObject(endpointIR.Request.PathParams)
	suffix := ""
	if pathCount > 0 {
		suffix = strconv.Itoa(pathCount + 1)
	}
	name := fmt.Sprintf("%s%sPath%s", method.VarName, svc.StructName, suffix)
	for j, arg := range params {
		patt := parameterAttributeByName(endpointIR.Request.PathParams, arg)
		att := makeHTTPType(patt)
		pointer := parameterPrimitivePointerByName(endpointIR.Request.PathParams, arg)
		if payloadPointer := payloadPrimitivePointerByName(endpointIR.Request.Payload, arg); payloadPointer {
			pointer = true
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
	code := renderPathInitCode(initArgs, pathParamsObj, expr.HTTPWildcardRegex.ReplaceAllString(path, "/%v"))
	return &InitData{
		Name:           name,
		Description:    fmt.Sprintf("%s returns the URL path to the %s service %s HTTP endpoint. ", name, svc.Name, method.Name),
		ServerArgs:     initArgs,
		ClientArgs:     initArgs,
		ReturnTypeName: "string",
		ReturnTypeRef:  "string",
		ServerCode:     code,
		ClientCode:     code,
	}
}

func (sds *ServicesData) buildRequirementSchemes(endpointIR *transportir.Endpoint) (service.RequirementsData, service.SchemesData, service.SchemesData, service.SchemesData, *service.SchemeData) {
	reqs, allSchemes := service.BuildRequirementsData(endpointIR.Security.Requirements, &expr.MethodExpr{Payload: endpointIR.Request.Payload})
	var (
		headerSchemes service.SchemesData
		bodySchemes   service.SchemesData
		querySchemes  service.SchemesData
		basicScheme   *service.SchemeData
	)
	for _, scheme := range allSchemes {
		switch scheme.Type {
		case "Basic":
			basicScheme = scheme
		default:
			switch scheme.In {
			case "query":
				querySchemes = querySchemes.Append(scheme)
			case "header":
				headerSchemes = headerSchemes.Append(scheme)
			default:
				bodySchemes = bodySchemes.Append(scheme)
			}
		}
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

func (sds *ServicesData) buildClientRequestInit(endpointIR *transportir.Endpoint, method *service.MethodData, svc *service.Data, routes []*RouteData) *InitData {
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
	if len(routes[0].PathInit.ClientArgs) > 0 && endpointIR.Request.Payload.Type != expr.Empty {
		payloadRef = svc.Scope.GoFullTypeRef(endpointIR.Request.Payload, pkg)
	}
	requestStruct := ""
	if endpointIR.Request.SkipBodyEncode {
		requestStruct = pkg + "." + method.RequestStruct
	}
	code := renderRequestInitCode(
		payloadRef,
		expr.IsObject(endpointIR.Request.Payload.Type),
		svc.Name,
		method.Name,
		args,
		routes[0].PathInit,
		routes[0].Verb,
		endpointIR.Stream.IsStreaming && !endpointIR.Stream.IsSSE,
		requestStruct,
	)
	return &InitData{
		Name:        name,
		Description: fmt.Sprintf("%s instantiates a HTTP request object with method and path set to call the %q service %q endpoint", name, svc.Name, method.Name),
		ClientCode:  code,
		ClientArgs:  []*InitArgData{{Ref: "v", AttributeData: &AttributeData{Name: "payload", VarName: "v", TypeRef: "any"}}},
	}
}

func (sds *ServicesData) initEndpointMultipartData(endpoint *EndpointData, endpointIR *transportir.Endpoint, method *service.MethodData, svc *service.Data) {
	if endpointIR.Request.Multipart && !endpoint.Payload.Request.MultipartGenerated {
		endpoint.MultipartRequestDecoder = &MultipartData{
			FuncName:    fmt.Sprintf("%s%sDecoderFunc", svc.StructName, method.VarName),
			InitName:    fmt.Sprintf("New%s%sDecoder", svc.StructName, method.VarName),
			VarName:     fmt.Sprintf("%s%sDecoderFn", svc.VarName, method.VarName),
			ServiceName: svc.Name,
			MethodName:  method.Name,
			Payload:     endpoint.Payload,
		}
	}
	if endpointIR.Request.Multipart {
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

func parameterAttributeByName(params []*transportir.Parameter, name string) *expr.AttributeExpr {
	for _, param := range params {
		if param.Name == name {
			return param.Attribute
		}
	}
	return nil
}

func parameterPrimitivePointerByName(params []*transportir.Parameter, name string) bool {
	for _, param := range params {
		if param.Name == name {
			return param.PrimitivePointer
		}
	}
	return false
}

func payloadPrimitivePointerByName(payload *expr.AttributeExpr, name string) bool {
	if payload == nil || !expr.IsObject(payload.Type) {
		return false
	}
	return payload.IsPrimitivePointer(name, true)
}

func pathParametersObject(params []*transportir.Parameter) *expr.Object {
	object := make(expr.Object, 0, len(params))
	for _, param := range params {
		object = append(object, &expr.NamedAttributeExpr{
			Name:      param.Name,
			Attribute: param.Attribute,
		})
	}
	return &object
}

func (sds *ServicesData) collectEndpointBodyAttributeTypes(endpointIR *transportir.Endpoint, sd *ServiceData) {
	unionBranchTypes := make(map[string]struct{})
	collectUnionBranchUserTypes(endpointIR.Request.RawBody, unionBranchTypes)
	if endpointIR.Stream.RequestPayload != nil && endpointIR.Stream.RequestPayload.Type != expr.Empty {
		collectUnionBranchUserTypes(endpointIR.Request.StreamingBody, unionBranchTypes)
	}

	appendTypeData := func(att *expr.AttributeExpr, ptr, server bool, target *[]*TypeData) {
		collectUserTypes(att.Type, func(ut expr.UserType) {
			if d := sds.attributeTypeData(ut, true, ptr, server, sd); d != nil {
				if !server && d.ValidateDef == "" {
					if _, ok := unionBranchTypes[ut.ID()]; ok {
						d.ValidateDef = "// no validations"
						d.ValidateRef = fmt.Sprintf("err = Validate%s(v)", d.VarName)
					}
				}
				*target = append(*target, d)
			}
		})
	}
	appendTypeData(endpointIR.Request.RawBody, true, true, &sd.ServerBodyAttributeTypes)
	appendTypeData(endpointIR.Request.RawBody, false, false, &sd.ClientBodyAttributeTypes)

	if endpointIR.Stream.RequestPayload != nil && endpointIR.Stream.RequestPayload.Type != expr.Empty {
		appendTypeData(endpointIR.Request.StreamingBody, true, true, &sd.ServerBodyAttributeTypes)
		appendTypeData(endpointIR.Request.StreamingBody, false, false, &sd.ClientBodyAttributeTypes)
	}

	if endpointIR.Response.Result != nil {
		for _, response := range endpointIR.Response.Responses {
			collectUserTypes(response.Body.Type, func(ut expr.UserType) {
				if d := sds.attributeTypeData(ut, false, true, false, sd); d != nil {
					sd.ClientBodyAttributeTypes = append(sd.ClientBodyAttributeTypes, d)
				}
			})
		}
	}
	for _, httpError := range endpointIR.Response.ErrorResponses {
		collectUserTypes(httpError.Body.Type, func(ut expr.UserType) {
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

func generatedMultipartRequestData(request *transportir.Request) (bool, []*MultipartFileFieldData) {
	if request == nil || !request.Multipart || request.Body == nil || request.Body.Type == expr.Empty || !expr.IsObject(request.Body.Type) {
		return false, nil
	}
	if !supportsGeneratedMultipartObject(request.Body) {
		return false, nil
	}
	fileFields := multipartFileFields(request.Body)
	if len(fileFields) == 1 {
		bodyObj := expr.AsObject(request.Body.Type)
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

func (sds *ServicesData) buildResultDataFromIR(endpointIR *transportir.Endpoint, sd *ServiceData) *ResultData {
	var (
		svc    = sd.Service
		ep     = svc.Method(endpointIR.MethodName)
		pkg    = pkgWithDefault(ep.ResultLoc, svc.PkgName)
		result = endpointIR.Response.Result
	)
	name, ref, view := buildResultMetadata(svc, result, pkg)
	responses, mustInit, result := sds.buildResultResponsesData(endpointIR, ep, result, sd)
	idAtt, idAttRequired := buildResultIDData(endpointIR.Response, result)
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

func buildResultMetadata(svc *service.Data, result *expr.AttributeExpr, pkg string) (string, string, string) {
	view := expr.DefaultView
	if v, ok := result.Meta.Last(expr.ViewMetaKey); ok {
		view = v
	}
	if result.Type == expr.Empty {
		return "", "", view
	}
	return svc.Scope.GoFullTypeName(result, pkg), svc.Scope.GoFullTypeRef(result, pkg), view
}

func (sds *ServicesData) buildResultResponsesData(
	endpointIR *transportir.Endpoint,
	ep *service.MethodData,
	result *expr.AttributeExpr,
	sd *ServiceData,
) ([]*ResponseData, bool, *expr.AttributeExpr) {
	viewed := false
	if ep.ViewedResult != nil {
		result = expr.AsObject(ep.ViewedResult.Type).Attribute("projected")
		viewed = true
	}
	responses := sds.buildResponsesFromIR(endpointIR, result, viewed, sd)
	mustInit := false
	for _, r := range responses {
		if len(r.ServerBody) > 0 || len(r.Headers) > 0 || len(r.Cookies) > 0 || r.TagName != "" {
			mustInit = true
			break
		}
	}
	return responses, mustInit, result
}

func buildResultIDData(response *transportir.Response, result *expr.AttributeExpr) (string, bool) {
	if response == nil || response.IDAttribute == "" {
		return "", false
	}
	return codegen.Goify(response.IDAttribute, true), result.IsRequired(response.IDAttribute)
}

func transportStringSlice(att *expr.AttributeExpr) bool {
	arr := expr.AsArray(att.Type)
	return arr != nil && arr.ElemType.Type.Kind() == expr.StringKind
}

func transportFieldBinding(name string, fieldAttr, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext) (string, expr.DataType, bool) {
	fieldType := svcAtt.Type
	if !expr.IsObject(svcAtt.Type) {
		return "", fieldType, false
	}
	svcField := svcAtt.Find(name)
	if svcField == nil {
		return "", fieldAttr.Type, false
	}
	fieldType = svcField.Type
	fieldName := codegen.GoifyAtt(fieldAttr, name, true)
	if svcCtx == nil {
		return fieldName, fieldType, svcAtt.IsPrimitivePointer(name, true)
	}
	return fieldName, fieldType, svcCtx.IsPrimitivePointer(name, svcAtt)
}

func (sds *ServicesData) buildTransportAttributeData(
	name string,
	attr *expr.AttributeExpr,
	required bool,
	pointer bool,
	fieldName string,
	fieldType expr.DataType,
	fieldPointer bool,
	validateCtx *codegen.AttributeContext,
	scope *codegen.NameScope,
) *AttributeData {
	varName := scope.Name(codegen.Goify(name, false))
	typeRef := scope.GoTypeRef(attr)
	if pointer {
		typeRef = "*" + typeRef
	}
	return &AttributeData{
		Name:         name,
		Description:  attr.Description,
		FieldName:    fieldName,
		FieldPointer: fieldPointer,
		FieldType:    fieldType,
		VarName:      varName,
		Required:     required,
		Type:         attr.Type,
		TypeName:     scope.GoTypeName(attr),
		TypeRef:      typeRef,
		Pointer:      pointer,
		Validate:     codegen.AttributeValidationCode(attr, nil, validateCtx, required, expr.IsAlias(attr.Type), varName, name),
		DefaultValue: attr.DefaultValue,
		Example:      attr.Example(sds.Root.API.ExampleGenerator),
	}
}

func (sds *ServicesData) buildTransportElement(
	name string,
	elem string,
	attr *expr.AttributeExpr,
	stringSlice bool,
	required bool,
	pointer bool,
	fieldName string,
	fieldType expr.DataType,
	fieldPointer bool,
	validateCtx *codegen.AttributeContext,
	scope *codegen.NameScope,
) *Element {
	return &Element{
		HTTPName:      elem,
		AttributeName: name,
		StringSlice:   stringSlice,
		Slice:         expr.AsArray(attr.Type) != nil,
		AttributeData: sds.buildTransportAttributeData(
			name,
			attr,
			required,
			pointer,
			fieldName,
			fieldType,
			fieldPointer,
			validateCtx,
			scope,
		),
	}
}

func (sds *ServicesData) extractHeaders(headersIR []*transportir.Header, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext, scope *codegen.NameScope) []*HeaderData {
	headers := make([]*HeaderData, 0, len(headersIR))
	for _, headerIR := range headersIR {
		name := headerIR.Name
		elem := headerIR.HTTPName
		var attr *expr.AttributeExpr
		if attr = svcAtt.Find(name); attr == nil {
			attr = svcAtt
		}
		stringSlice := transportStringSlice(attr)
		hattr := makeHTTPType(attr)
		pointer := headerIR.PrimitivePointer
		fieldName, fieldType, fieldPointer := transportFieldBinding(name, attr, svcAtt, svcCtx)
		headers = append(headers, &HeaderData{
			CanonicalName: http.CanonicalHeaderKey(elem),
			Element:       sds.buildTransportElement(name, elem, hattr, stringSlice, headerIR.Required, pointer, fieldName, fieldType, fieldPointer, svcCtx, scope),
		})
	}
	return headers
}

func (sds *ServicesData) extractResponseCookies(cookiesIR []*transportir.Cookie, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext, scope *codegen.NameScope) []*CookieData {
	cookies := make([]*CookieData, 0, len(cookiesIR))
	for _, cookieIR := range cookiesIR {
		name := cookieIR.Name
		if name == "" {
			continue
		}
		cookie := sds.cookieData(name, cookieIR.HTTPName, cookieIR.Required, cookieIR.PrimitivePointer, cookieIR.Attribute, svcAtt, svcCtx, scope)
		cookie.MaxAge = cookieIR.MaxAge
		cookie.Path = cookieIR.Path
		cookie.Domain = cookieIR.Domain
		cookie.Secure = cookieIR.Secure
		cookie.HTTPOnly = cookieIR.HTTPOnly
		switch cookieIR.SameSite {
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

func (sds *ServicesData) cookieData(name, elem string, required bool, pointer bool, mappedAttr *expr.AttributeExpr, svcAtt *expr.AttributeExpr, svcCtx *codegen.AttributeContext, scope *codegen.NameScope) *CookieData {
	var hattr *expr.AttributeExpr
	if hattr = svcAtt.Find(name); hattr == nil {
		if mappedAttr != nil {
			hattr = mappedAttr
		} else {
			hattr = svcAtt
		}
	}
	stringSlice := transportStringSlice(hattr)
	hattr = makeHTTPType(hattr)
	fieldName, fieldType, fieldPointer := transportFieldBinding(name, hattr, svcAtt, svcCtx)
	return &CookieData{
		Element: sds.buildTransportElement(name, elem, hattr, stringSlice, required, pointer, fieldName, fieldType, fieldPointer, svcCtx, scope),
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
	var example any
	if sds != nil && sds.Root != nil && sds.Root.API != nil {
		example = att.Example(sds.Root.API.ExampleGenerator)
	}
	return &TypeData{
		Name:        ut.Name(),
		VarName:     name,
		Description: desc,
		Def:         goTypeDef(rd.Scope, ut.Attribute(), ptr, hctx.UseDefault),
		Ref:         rd.Scope.GoTypeRef(att),
		ValidateDef: validate,
		ValidateRef: validateRef,
		Example:     example,
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
// NeedDialer returns true if at least one method in the defined services
// uses WebSocket for sending payload or result.
func NeedDialer(data []*ServiceData) bool {
	return slices.ContainsFunc(data, HasWebSocket)
}
