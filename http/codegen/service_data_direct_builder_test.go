package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestHTTPDirectBuilderSeams(t *testing.T) {
	t.Run("buildEndpointData preserves mixed result assembly", func(t *testing.T) {
		services, endpointExpr, svcData := firstHTTPBuildContext(t, testdata.MixedResultsDSL)
		endpointIR := transportir.BuildEndpoint(endpointExpr)

		endpoint := services.buildEndpointDataFromIR(endpointIR, svcData.Service, svcData, codegen.NewNameScope())
		require.True(t, endpoint.HasMixedResults)
		require.NotNil(t, endpoint.SSE)
		require.Equal(t, "EncodeCreateRequest", endpoint.RequestEncoder)
		require.Equal(t, "BuildCreateRequest", endpoint.RequestInit.Name)
	})

	t.Run("buildPathInitData keeps wildcard arguments in route order", func(t *testing.T) {
		services, endpointExpr, svcData := firstHTTPBuildContext(t, testdata.PathMultipleParamsDSL)
		endpointIR := transportir.BuildEndpoint(endpointExpr)
		method := svcData.Service.Method(endpointIR.MethodName)
		require.NotNil(t, method)

		pathInit := services.buildPathInitData(endpointIR, method, svcData.Service, svcData, endpointIR.Routes[0].Path, 0)
		require.Contains(t, pathInit.Name, method.VarName)
		require.Contains(t, pathInit.Name, svcData.Service.StructName)
		require.Contains(t, pathInit.Name, "Path")
		require.Len(t, pathInit.ClientArgs, 2)
		require.Equal(t, "a", pathInit.ClientArgs[0].Name)
		require.Equal(t, "b", pathInit.ClientArgs[1].Name)
		require.Contains(t, pathInit.ClientCode, `fmt.Sprintf("/one/%v/two/%v/three"`)
	})

	t.Run("buildRequirementSchemes partitions transports by location", func(t *testing.T) {
		t.Run("query bearer token", func(t *testing.T) {
			services, endpointExpr, _ := firstHTTPBuildContext(t, testdata.PayloadJWTAuthorizationQueryDSL)
			endpointIR := transportir.BuildEndpoint(endpointExpr)

			reqs, headerSchemes, bodySchemes, querySchemes, basicScheme := services.buildRequirementSchemes(endpointIR)
			require.NotEmpty(t, reqs)
			require.Empty(t, headerSchemes)
			require.Empty(t, bodySchemes)
			require.Len(t, querySchemes, 1)
			require.Equal(t, "query", querySchemes[0].In)
			require.Nil(t, basicScheme)
		})

		t.Run("custom header bearer token", func(t *testing.T) {
			services, endpointExpr, _ := firstHTTPBuildContext(t, testdata.PayloadJWTAuthorizationCustomHeaderDSL)
			endpointIR := transportir.BuildEndpoint(endpointExpr)

			reqs, headerSchemes, bodySchemes, querySchemes, basicScheme := services.buildRequirementSchemes(endpointIR)
			require.NotEmpty(t, reqs)
			require.Len(t, headerSchemes, 1)
			require.Equal(t, "header", headerSchemes[0].In)
			require.Empty(t, bodySchemes)
			require.Empty(t, querySchemes)
			require.Nil(t, basicScheme)
		})
	})

	t.Run("buildPayloadData projects jsonrpc ids", func(t *testing.T) {
		services, endpointExpr, svcData := firstJSONRPCBuildContext(t, jsonrpcIDProjectionDSL)
		endpointIR := transportir.BuildEndpoint(endpointExpr)

		payload := services.buildPayloadDataFromIR(endpointIR, svcData)
		require.Equal(t, "ID", payload.IDAttribute)
		require.True(t, payload.IDAttributeRequired)
		require.NotNil(t, payload.Request)
		require.NotNil(t, payload.Request.PayloadInit)
		require.Empty(t, payload.DecoderReturnValue)
	})

	t.Run("buildEndpointData wires multipart encoder and decoder helpers", func(t *testing.T) {
		services, endpointExpr, svcData := firstHTTPBuildContext(t, testdata.PayloadMultipartPrimitiveDSL)
		endpointIR := transportir.BuildEndpoint(endpointExpr)

		endpoint := services.buildEndpointDataFromIR(endpointIR, svcData.Service, svcData, codegen.NewNameScope())
		require.NotNil(t, endpoint.MultipartRequestDecoder)
		require.NotNil(t, endpoint.MultipartRequestEncoder)
		require.Equal(t, "NewServiceMultipartPrimitiveMethodMultipartPrimitiveDecoder", endpoint.MultipartRequestDecoder.InitName)
		require.Equal(t, "NewServiceMultipartPrimitiveMethodMultipartPrimitiveEncoder", endpoint.MultipartRequestEncoder.InitName)
	})

	t.Run("buildResultData keeps default view and jsonrpc ids", func(t *testing.T) {
		services, endpointExpr, svcData := firstJSONRPCBuildContext(t, jsonrpcIDProjectionDSL)
		endpointIR := transportir.BuildEndpoint(endpointExpr)

		result := services.buildResultDataFromIR(endpointIR, svcData)
		require.Equal(t, expr.DefaultView, result.View)
		require.Equal(t, "ID", result.IDAttribute)
		require.True(t, result.IDAttributeRequired)
		require.True(t, result.MustInit)
		require.NotEmpty(t, result.Responses)
	})

	t.Run("buildRequestBodyType flattens form union helper field", func(t *testing.T) {
		services, endpointExpr, svcData := firstHTTPBuildContext(t, testdata.PayloadFormBodyUnionDSL)

		bodyType := services.buildRequestBodyType(endpointExpr.Body, endpointExpr.MethodExpr.Payload, endpointExpr.Name(), endpointExpr.FormRequest, false, svcData)
		require.NotNil(t, bodyType)
		require.Equal(t, "Values", bodyType.FlatFormUnionField)
		require.True(t, bodyType.FlatFormUnionPointer)
		require.Equal(t, "type", bodyType.FlatFormUnionTypeKey)
		require.Equal(t, "*Values", bodyType.FlatFormUnionRef)
		require.NotNil(t, bodyType.Init)
		require.NotEmpty(t, bodyType.Init.ClientCode)
	})

	t.Run("buildRequestBodyType only emits constructors on the client", func(t *testing.T) {
		services, endpointExpr, svcData := firstHTTPBuildContext(t, testdata.PayloadFormBodyUnionDSL)

		clientBodyType := services.buildRequestBodyType(endpointExpr.Body, endpointExpr.MethodExpr.Payload, endpointExpr.Name(), endpointExpr.FormRequest, false, svcData)
		serverBodyType := services.buildRequestBodyType(endpointExpr.Body, endpointExpr.MethodExpr.Payload, endpointExpr.Name(), endpointExpr.FormRequest, true, svcData)
		require.NotNil(t, clientBodyType)
		require.NotNil(t, clientBodyType.Init)
		require.NotNil(t, serverBodyType)
		require.Nil(t, serverBodyType.Init)
	})

	t.Run("buildResponseBodyType keeps projected view names", func(t *testing.T) {
		services, endpointExpr, svcData := firstHTTPBuildContext(t, testdata.ResultWithResultViewDSL)
		method := svcData.Service.Method(endpointExpr.Name())

		bodyType := services.buildResponseBodyType(endpointExpr.Responses[0].Body, endpointExpr.MethodExpr.Result, method.ResultLoc, endpointExpr.Name(), true, stringPtr("full"), svcData)
		require.NotNil(t, bodyType)
		require.Equal(t, "full", bodyType.View)
		require.NotNil(t, bodyType.Init)
		require.NotEmpty(t, bodyType.Init.ServerCode)
	})

	t.Run("buildResponseBodyType only emits constructors on the server", func(t *testing.T) {
		services, endpointExpr, svcData := firstHTTPBuildContext(t, testdata.ResultBodyCollectionDSL)
		method := svcData.Service.Method(endpointExpr.Name())

		serverBodyType := services.buildResponseBodyType(endpointExpr.Responses[0].Body, endpointExpr.MethodExpr.Result, method.ResultLoc, endpointExpr.Name(), true, nil, svcData)
		clientBodyType := services.buildResponseBodyType(endpointExpr.Responses[0].Body, endpointExpr.MethodExpr.Result, method.ResultLoc, endpointExpr.Name(), false, nil, svcData)
		require.NotNil(t, serverBodyType)
		require.NotNil(t, serverBodyType.Init)
		require.NotNil(t, clientBodyType)
		require.Nil(t, clientBodyType.Init)
	})

	t.Run("buildResponseBodyType uses endpoint scoped wrapper for collection bodies", func(t *testing.T) {
		services, endpointExpr, svcData := firstHTTPBuildContext(t, testdata.ResultBodyCollectionDSL)
		method := svcData.Service.Method(endpointExpr.Name())

		bodyType := services.buildResponseBodyType(endpointExpr.Responses[0].Body, endpointExpr.MethodExpr.Result, method.ResultLoc, endpointExpr.Name(), true, nil, svcData)
		require.NotNil(t, bodyType)
		require.Equal(t, "MethodBodyCollectionResponseBody", bodyType.Name)
		require.Equal(t, "MethodBodyCollectionResponseBody", bodyType.VarName)
		require.NotNil(t, bodyType.Init)
		require.Equal(t, "NewMethodBodyCollectionResponseBody", bodyType.Init.Name)
	})

	t.Run("mixed transport graph stays split across http and jsonrpc service data", func(t *testing.T) {
		root := RunHTTPDSL(t, mixedTransportServiceDataDSL)

		httpServices := CreateHTTPServices(root)
		plain := httpServices.Get("PlainHTTP")
		require.NotNil(t, plain)
		require.Len(t, plain.Endpoints, 1)
		require.NotNil(t, plain.Endpoints[0].Payload.Request.ServerBody)
		require.Equal(t, "EncodeCreateRequest", plain.Endpoints[0].RequestEncoder)

		jsonrpcServices := NewServicesData(service.NewServicesData(root), &root.API.JSONRPC.HTTPExpr)
		post := jsonrpcServices.Get("RPCPost")
		require.NotNil(t, post)
		require.Equal(t, "ID", post.Endpoints[0].Payload.IDAttribute)
		require.Equal(t, "ID", post.Endpoints[0].Result.IDAttribute)

		sse := jsonrpcServices.Get("RPCSSE")
		require.NotNil(t, sse)
		require.True(t, sse.Endpoints[0].HasMixedResults)
		require.NotNil(t, sse.Endpoints[0].SSE)
		require.Equal(t, "POST", sse.Endpoints[0].Routes[0].Verb)

		ws := jsonrpcServices.Get("RPCWebSocket")
		require.NotNil(t, ws)
		require.NotNil(t, ws.Endpoints[0].ServerWebSocket)
		require.Equal(t, "GET", ws.Endpoints[0].Routes[0].Verb)
	})

	t.Run("union collections are discovered across HTTP transport paths", func(t *testing.T) {
		root := RunHTTPDSL(t, mixedTransportServiceDataDSL)
		services := CreateHTTPServices(root)
		svc := services.Get("UnionHTTPPaths")
		require.NotNil(t, svc)

		require.ElementsMatch(t, []string{
			"ErrorChoice",
			"RequestChoice",
			"ResponseChoice",
			"StreamChoice",
		}, httpUnionTypeNames(svc.UnionTypes))
	})

	t.Run("buildErrorsData uses IR-owned error responses", func(t *testing.T) {
		services, endpointExpr, svcData := firstHTTPBuildContext(t, responseInitArgsDSL)
		endpointIR := transportir.BuildEndpoint(endpointExpr)
		require.NotNil(t, endpointIR.Response)
		require.NotEmpty(t, endpointIR.Response.ErrorResponses)

		errorIR := endpointIR.Response.ErrorResponses[0]
		errors := services.buildErrorsDataFromIR(endpointIR, svcData)
		require.Len(t, errors, 1)
		require.Len(t, errors[0].Errors, 1)

		errResp := errors[0].Errors[0].Response
		require.NotNil(t, errResp)
		require.Equal(t, statusCodeToHTTPConst(errorIR.StatusCode), errors[0].StatusCode)
		require.Equal(t, statusCodeToHTTPConst(errorIR.StatusCode), errResp.StatusCode)
		require.Equal(t, errorIR.StatusCode, errResp.Code)
		require.Equal(t, errorIR.Headers[0].HTTPName, errResp.Headers[0].HTTPName)
		require.Equal(t, errorIR.Cookies[0].HTTPName, errResp.Cookies[0].HTTPName)
		require.NotNil(t, errResp.ClientBody)
		require.Equal(t, errorIR.ContentType, errResp.ContentType)
	})

	t.Run("websocket payload init stays available through IR service data", func(t *testing.T) {
		endpoint := firstEndpointData(t, testdata.StreamingAliasedArrayDSL)
		require.NotNil(t, endpoint.ClientWebSocket)
		require.NotNil(t, endpoint.ClientWebSocket.Payload)
		require.NotNil(t, endpoint.ClientWebSocket.Payload.Init)
		require.Equal(t, "NewStreamStreamingBody", endpoint.ClientWebSocket.Payload.Init.Name)
	})
}

func firstEndpointData(t *testing.T, dsl func()) *EndpointData {
	t.Helper()

	svc := firstServiceData(t, dsl)
	require.NotEmpty(t, svc.Endpoints)
	return svc.Endpoints[0]
}

func firstServiceData(t *testing.T, dsl func()) *ServiceData {
	t.Helper()

	root := RunHTTPDSL(t, dsl)
	require.NotEmpty(t, root.API.HTTP.Services)

	services := CreateHTTPServices(root)
	svc := services.Get(root.API.HTTP.Services[0].Name())
	require.NotNil(t, svc)
	return svc
}

func firstHTTPBuildContext(t *testing.T, dsl func()) (*ServicesData, *expr.HTTPEndpointExpr, *ServiceData) {
	t.Helper()

	root := RunHTTPDSL(t, dsl)
	require.NotEmpty(t, root.API.HTTP.Services)

	services := CreateHTTPServices(root)
	httpSvc := root.API.HTTP.Services[0]
	require.NotEmpty(t, httpSvc.HTTPEndpoints)

	svc := services.Get(httpSvc.Name())
	require.NotNil(t, svc)
	return services, httpSvc.HTTPEndpoints[0], svc
}

func firstJSONRPCEndpointData(t *testing.T, dsl func()) *EndpointData {
	t.Helper()

	svc := firstJSONRPCServiceData(t, dsl)
	require.NotEmpty(t, svc.Endpoints)
	return svc.Endpoints[0]
}

func firstJSONRPCServiceData(t *testing.T, dsl func()) *ServiceData {
	t.Helper()

	root := RunHTTPDSL(t, dsl)
	require.NotEmpty(t, root.API.JSONRPC.Services)

	services := NewServicesData(service.NewServicesData(root), &root.API.JSONRPC.HTTPExpr)
	svc := services.Get(root.API.JSONRPC.Services[0].Name())
	require.NotNil(t, svc)
	return svc
}

func firstJSONRPCBuildContext(t *testing.T, dsl func()) (*ServicesData, *expr.HTTPEndpointExpr, *ServiceData) {
	t.Helper()

	root := RunHTTPDSL(t, dsl)
	require.NotEmpty(t, root.API.JSONRPC.Services)

	services := NewServicesData(service.NewServicesData(root), &root.API.JSONRPC.HTTPExpr)
	httpSvc := root.API.JSONRPC.Services[0]
	require.NotEmpty(t, httpSvc.HTTPEndpoints)

	svc := services.Get(httpSvc.Name())
	require.NotNil(t, svc)
	return services, httpSvc.HTTPEndpoints[0], svc
}

func lastQueryParam(t *testing.T, endpoint *EndpointData) *ParamData {
	t.Helper()

	require.NotEmpty(t, endpoint.Payload.Request.QueryParams)
	return endpoint.Payload.Request.QueryParams[len(endpoint.Payload.Request.QueryParams)-1]
}

func initArgNames(args []*InitArgData) []string {
	names := make([]string, len(args))
	for i, arg := range args {
		names[i] = arg.Name
	}
	return names
}

func initArgFieldNames(args []*InitArgData) []string {
	names := make([]string, len(args))
	for i, arg := range args {
		names[i] = arg.FieldName
	}
	return names
}

func initArgRefs(args []*InitArgData) []string {
	refs := make([]string, len(args))
	for i, arg := range args {
		refs[i] = arg.Ref
	}
	return refs
}

func bodyViews(types []*TypeData) []string {
	views := make([]string, len(types))
	for i, td := range types {
		views[i] = td.View
	}
	return views
}

func httpUnionTypeNames(types []*service.UnionTypeData) []string {
	names := make([]string, len(types))
	for i, union := range types {
		names[i] = union.Name
	}
	return names
}

func stringPtr(v string) *string {
	return &v
}

func requestEncoderBodyDSL() {
	Service("RequestEncoderBody", func() {
		Method("show", func() {
			Payload(func() {
				Attribute("name", String)
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}

func responseInitArgsDSL() {
	var successBody = Type("ResponseInitArgsSuccessBody", func() {
		Attribute("value", String)
	})
	var errorBody = Type("ResponseInitArgsErrorBody", func() {
		Attribute("message", String)
	})
	var methodError = Type("ResponseInitArgsError", func() {
		Attribute("body", errorBody)
		Attribute("code", Int)
		Attribute("session", String)
		Required("body", "code", "session")
	})

	Service("ResponseInitArgs", func() {
		Method("show", func() {
			Result(func() {
				Attribute("body", successBody)
				Attribute("header", String)
				Attribute("session", String)
				Required("body", "header", "session")
			})
			Error("bad_request", methodError)
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Body("body")
					Header("header:X-Header")
					Cookie("session:session")
				})
				Response("bad_request", StatusBadRequest, func() {
					Body("body")
					Header("code:X-Code")
					Cookie("session:session")
				})
			})
		})
	})
}

func jsonrpcIDProjectionDSL() {
	Service("JSONRPCIDProjection", func() {
		Method("show", func() {
			Payload(func() {
				ID("id", String)
				Attribute("query", String)
				Required("id")
			})
			Result(func() {
				ID("id", String)
				Attribute("value", String)
				Required("id")
			})
			JSONRPC(func() {})
		})
		JSONRPC(func() {
			Path("/rpc")
		})
	})
}

func responseTaglessFirstDSL() {
	Service("ResponseTaglessFirst", func() {
		Method("show", func() {
			Result(func() {
				Attribute("h", String)
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK)
				Response(StatusAccepted, func() {
					Header("h")
					Tag("h", "value")
				})
			})
		})
	})
}

func mixedTransportServiceDataDSL() {
	API("mixed-transport-service-data", func() {
		JSONRPC(func() {})
	})

	Service("PlainHTTP", func() {
		Method("create", func() {
			Payload(func() {
				Attribute("id", String)
				Attribute("name", String)
				Required("id", "name")
			})
			Result(String)
			HTTP(func() {
				POST("/plain/{id}")
				Body("name")
			})
		})
	})

	Service("RPCPost", func() {
		JSONRPC(func() {
			POST("/rpc")
		})
		Method("ping", func() {
			Payload(func() {
				ID("id", String)
			})
			Result(func() {
				ID("id", String)
				Attribute("value", String)
			})
			JSONRPC(func() {})
		})
	})

	Service("RPCSSE", func() {
		JSONRPC(func() {
			POST("/events")
		})
		Method("events/stream", func() {
			Payload(func() {
				ID("id", String)
				Attribute("filter", String)
			})
			Result(func() {
				Attribute("accepted", Boolean)
			})
			StreamingResult(func() {
				Attribute("event", String)
			})
			JSONRPC(func() {
				ServerSentEvents()
			})
		})
	})

	Service("RPCWebSocket", func() {
		JSONRPC(func() {
			GET("/ws")
		})
		Method("stream", func() {
			StreamingPayload(func() {
				ID("id", String)
				Attribute("message", String)
			})
			StreamingResult(func() {
				Attribute("event", String)
			})
			JSONRPC(func() {})
		})
	})

	var requestEnvelope = Type("HTTPRequestUnionEnvelope", func() {
		OneOf("request_choice", func() {
			Attribute("name", String)
			Attribute("count", Int)
		})
	})
	var streamEnvelope = Type("HTTPStreamUnionEnvelope", func() {
		OneOf("stream_choice", func() {
			Attribute("message", String)
			Attribute("sequence", Int)
		})
	})
	var responseEnvelope = Type("HTTPResponseUnionEnvelope", func() {
		OneOf("response_choice", func() {
			Attribute("accepted", Boolean)
			Attribute("location", String)
		})
	})
	var errorEnvelope = Type("HTTPErrorUnionEnvelope", func() {
		OneOf("error_choice", func() {
			Attribute("field", String)
			Attribute("retry_after", Int)
		})
	})

	Service("UnionHTTPPaths", func() {
		Method("request", func() {
			Payload(ArrayOf(requestEnvelope))
			HTTP(func() {
				POST("/request")
			})
		})
		Method("stream", func() {
			StreamingPayload(MapOf(String, streamEnvelope))
			Result(String)
			HTTP(func() {
				GET("/stream")
			})
		})
		Method("response", func() {
			Result(ArrayOf(responseEnvelope))
			HTTP(func() {
				GET("/response")
				Response(StatusOK)
			})
		})
		Method("failure", func() {
			Error("invalid", MapOf(String, errorEnvelope))
			HTTP(func() {
				POST("/failure")
				Response("invalid", StatusBadRequest)
			})
		})
	})
}
