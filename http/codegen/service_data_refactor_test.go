package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/codegen/service"
	. "goa.design/goa/v3/dsl"
	"goa.design/goa/v3/expr"
	"goa.design/goa/v3/http/codegen/testdata"
)

func TestHTTPPayloadDecoderReturnValueFallback(t *testing.T) {
	cases := []struct {
		name string
		dsl  func()
		want string
	}{
		{
			name: "path params",
			dsl:  testdata.PayloadPathPrimitiveStringValidateDSL,
			want: "p",
		},
		{
			name: "headers",
			dsl:  testdata.PayloadHeaderPrimitiveStringValidateDSL,
			want: "h",
		},
		{
			name: "cookies",
			dsl:  testdata.PayloadCookiePrimitiveStringValidateDSL,
			want: "c",
		},
		{
			name: "whole map query payload",
			dsl:  testdata.PayloadMapQueryPrimitivePrimitiveDSL,
			want: "query",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			endpoint := firstEndpointData(t, c.dsl)
			require.Equal(t, c.want, endpoint.Payload.DecoderReturnValue)
		})
	}
}

func TestHTTPMapQueryParamShape(t *testing.T) {
	cases := []struct {
		name             string
		dsl              func()
		wantHTTPName     string
		wantFieldName    string
		wantVarName      string
		wantRequired     bool
		wantMapPayload   bool
		wantMapQueryName string
	}{
		{
			name:             "whole payload map query",
			dsl:              testdata.PayloadMapQueryPrimitivePrimitiveDSL,
			wantHTTPName:     "query",
			wantFieldName:    "",
			wantVarName:      "query",
			wantRequired:     true,
			wantMapPayload:   true,
			wantMapQueryName: "",
		},
		{
			name:             "named object field map query",
			dsl:              testdata.PayloadMapQueryObjectDSL,
			wantHTTPName:     "c",
			wantFieldName:    "C",
			wantVarName:      "c",
			wantRequired:     true,
			wantMapPayload:   false,
			wantMapQueryName: "c",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			endpoint := firstEndpointData(t, c.dsl)
			query := lastQueryParam(t, endpoint)

			require.NotNil(t, query.MapQueryParams)
			require.Equal(t, c.wantMapQueryName, *query.MapQueryParams)
			require.Equal(t, c.wantMapPayload, query.Map)
			require.Equal(t, c.wantHTTPName, query.HTTPName)
			require.Equal(t, c.wantFieldName, query.FieldName)
			require.Equal(t, c.wantVarName, query.VarName)
			require.Equal(t, c.wantRequired, query.Required)
		})
	}
}

func TestHTTPRequestEncoderSelection(t *testing.T) {
	cases := []struct {
		name           string
		dsl            func()
		expectEncoder  bool
		expectBodyInit bool
	}{
		{
			name:           "no payload",
			dsl:            testdata.MultiNoPayloadDSL,
			expectEncoder:  false,
			expectBodyInit: false,
		},
		{
			name:           "header payload",
			dsl:            testdata.PayloadHeaderStringDSL,
			expectEncoder:  true,
			expectBodyInit: false,
		},
		{
			name:           "body payload",
			dsl:            requestEncoderBodyDSL,
			expectEncoder:  true,
			expectBodyInit: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			endpoint := firstEndpointData(t, c.dsl)
			if c.expectEncoder {
				require.NotEmpty(t, endpoint.RequestEncoder)
			} else {
				require.Empty(t, endpoint.RequestEncoder)
			}
			if c.expectBodyInit {
				require.NotNil(t, endpoint.RequestInit)
			}
		})
	}
}

func TestHTTPPayloadRequestValidationTriggers(t *testing.T) {
	cases := []struct {
		name string
		dsl  func()
	}{
		{
			name: "path param validation",
			dsl:  testdata.PayloadPathStringValidateDSL,
		},
		{
			name: "query param validation",
			dsl:  testdata.PayloadQueryBoolValidateDSL,
		},
		{
			name: "header validation",
			dsl:  testdata.PayloadHeaderStringValidateDSL,
		},
		{
			name: "cookie validation",
			dsl:  testdata.PayloadCookieStringValidateDSL,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			endpoint := firstEndpointData(t, c.dsl)
			require.True(t, endpoint.Payload.Request.MustValidate)
		})
	}
}

func TestHTTPResponsesMoveTaglessCaseLast(t *testing.T) {
	endpoint := firstEndpointData(t, responseTaglessFirstDSL)
	require.Len(t, endpoint.Result.Responses, 2)
	require.Equal(t, "H", endpoint.Result.Responses[0].TagName)
	require.Equal(t, "value", endpoint.Result.Responses[0].TagValue)
	require.Empty(t, endpoint.Result.Responses[1].TagName)
}

func TestHTTPMultipartEncoderDecoderGating(t *testing.T) {
	cases := []struct {
		name               string
		dsl                func()
		expectDecoder      bool
		expectGenerated    bool
		expectFileFieldLen int
	}{
		{
			name:               "custom multipart decoder required",
			dsl:                testdata.PayloadMultipartPrimitiveDSL,
			expectDecoder:      true,
			expectGenerated:    false,
			expectFileFieldLen: 0,
		},
		{
			name:               "generated multipart object",
			dsl:                testdata.PayloadMultipartObjectGeneratedDSL,
			expectDecoder:      false,
			expectGenerated:    true,
			expectFileFieldLen: 1,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			endpoint := firstEndpointData(t, c.dsl)
			require.NotNil(t, endpoint.MultipartRequestEncoder)
			if c.expectDecoder {
				require.NotNil(t, endpoint.MultipartRequestDecoder)
			} else {
				require.Nil(t, endpoint.MultipartRequestDecoder)
			}
			require.Equal(t, c.expectGenerated, endpoint.Payload.Request.MultipartGenerated)
			require.Len(t, endpoint.Payload.Request.MultipartFileFields, c.expectFileFieldLen)
		})
	}
}

func TestHTTPBuildStreamPayloadMultipartGating(t *testing.T) {
	cases := []struct {
		name                 string
		dsl                  func()
		expectBuildStreaming bool
	}{
		{
			name:                 "skip request body encode decode",
			dsl:                  testdata.SkipRequestBodyEncodeDecodeDSL,
			expectBuildStreaming: true,
		},
		{
			name:                 "custom multipart decoder path",
			dsl:                  testdata.PayloadMultipartPrimitiveDSL,
			expectBuildStreaming: false,
		},
		{
			name:                 "generated multipart object path",
			dsl:                  testdata.PayloadMultipartObjectGeneratedDSL,
			expectBuildStreaming: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			endpoint := firstEndpointData(t, c.dsl)
			if c.expectBuildStreaming {
				require.NotEmpty(t, endpoint.BuildStreamPayload)
				return
			}
			require.Empty(t, endpoint.BuildStreamPayload)
		})
	}
}

func TestHTTPFileServerPathNormalizationAndWildcardExtraction(t *testing.T) {
	cases := []struct {
		name             string
		dsl              func()
		wantRequestPaths []string
		wantDirFlags     []bool
		wantPathParams   []string
	}{
		{
			name:             "service root and wildcard paths",
			dsl:              testdata.ServerMultipleFilesDSL,
			wantRequestPaths: []string{"/file.json", "/", "/file.json", "/"},
			wantDirFlags:     []bool{false, false, false, true},
			wantPathParams:   []string{"", "", "", "wildcard"},
		},
		{
			name:             "prefixed root and wildcard paths",
			dsl:              testdata.ServerMultipleFilesWithPrefixPathDSL,
			wantRequestPaths: []string{"/server_file_server/file.json", "/server_file_server", "/server_file_server/file.json", "/server_file_server"},
			wantDirFlags:     []bool{false, false, false, true},
			wantPathParams:   []string{"", "", "", "wildcard"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := firstServiceData(t, c.dsl)
			require.Len(t, svc.FileServers, len(c.wantRequestPaths))
			for i, fs := range svc.FileServers {
				require.Equal(t, []string{c.wantRequestPaths[i]}, fs.RequestPaths)
				require.Equal(t, c.wantDirFlags[i], fs.IsDir)
				require.Equal(t, c.wantPathParams[i], fs.PathParam)
			}
		})
	}
}

func TestHTTPErrorResponseContentTypeSuppression(t *testing.T) {
	cases := []struct {
		name string
		dsl  func()
		want string
	}{
		{
			name: "default error response suppresses sentinel",
			dsl:  testdata.DefaultErrorResponseDSL,
			want: "",
		},
		{
			name: "explicit error response content type preserved",
			dsl:  testdata.DefaultErrorResponseWithContentTypeDSL,
			want: "application/xml",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			endpoint := firstEndpointData(t, c.dsl)
			require.NotEmpty(t, endpoint.Errors)
			require.NotEmpty(t, endpoint.Errors[0].Errors)
			require.NotNil(t, endpoint.Errors[0].Errors[0].Response)
			require.Equal(t, c.want, endpoint.Errors[0].Errors[0].Response.ContentType)
		})
	}
}

func TestHTTPResultAndErrorInitArgAssembly(t *testing.T) {
	endpoint := firstEndpointData(t, responseInitArgsDSL)

	require.NotNil(t, endpoint.Result)
	require.NotEmpty(t, endpoint.Result.Responses)
	resultInit := endpoint.Result.Responses[0].ResultInit
	require.NotNil(t, resultInit)
	require.Equal(t, "Body", resultInit.ReturnTypeAttribute)
	require.Equal(t, []string{"body", "header", "session"}, initArgNames(resultInit.ClientArgs))
	require.Equal(t, []string{"", "Header", "Session"}, initArgFieldNames(resultInit.ClientArgs))
	require.Equal(t, []string{"&body", "header", "session"}, initArgRefs(resultInit.ClientArgs))

	require.NotEmpty(t, endpoint.Errors)
	require.NotEmpty(t, endpoint.Errors[0].Errors)
	errorInit := endpoint.Errors[0].Errors[0].Response.ResultInit
	require.NotNil(t, errorInit)
	require.Equal(t, "Body", errorInit.ReturnTypeAttribute)
	require.Equal(t, []string{"body", "code", "session"}, initArgNames(errorInit.ClientArgs))
	require.Equal(t, []string{"", "Code", "Session"}, initArgFieldNames(errorInit.ClientArgs))
	require.Equal(t, []string{"&body", "code", "session"}, initArgRefs(errorInit.ClientArgs))
}

func TestHTTPJSONRPCIDProjection(t *testing.T) {
	endpoint := firstJSONRPCEndpointData(t, jsonrpcIDProjectionDSL)

	require.NotNil(t, endpoint.Payload)
	require.Equal(t, "ID", endpoint.Payload.IDAttribute)
	require.True(t, endpoint.Payload.IDAttributeRequired)

	require.NotNil(t, endpoint.Result)
	require.Equal(t, "ID", endpoint.Result.IDAttribute)
	require.True(t, endpoint.Result.IDAttributeRequired)
}

func TestHTTPClientRequestInitUsesServiceTypeRefsForAliasedPathParams(t *testing.T) {
	svc := firstServiceData(t, testdata.PathIntAliasDSL)
	require.NotEmpty(t, svc.Endpoints)
	endpoint := svc.Endpoints[0]
	require.NotEmpty(t, endpoint.Routes)
	require.NotNil(t, endpoint.RequestInit)

	pathInit := endpoint.Routes[0].PathInit
	require.NotNil(t, pathInit)
	require.Len(t, pathInit.ClientArgs, 3)

	for _, arg := range pathInit.ClientArgs {
		require.True(t, arg.IsAliased)
		require.Equal(t, svc.Scope.GoTypeRef(&expr.AttributeExpr{Type: arg.Type}), arg.ServiceTypeRef)
	}
}

func TestHTTPResponseBodyGenerationCoverage(t *testing.T) {
	t.Run("inline response body keeps projected body without synthetic origin attribute", func(t *testing.T) {
		endpoint := firstEndpointData(t, testdata.ExplicitBodyUserResultObjectDSL)
		require.NotNil(t, endpoint.Result)
		require.NotEmpty(t, endpoint.Result.Responses)
		resp := endpoint.Result.Responses[0]
		require.Empty(t, resp.ResultAttr)
		require.Len(t, resp.ServerBody, 1)
		require.NotNil(t, resp.ClientBody)
	})

	t.Run("Body(\"a\") keeps result attr for projected attribute bodies", func(t *testing.T) {
		endpoint := firstEndpointData(t, testdata.ExplicitBodyUserResultMultipleViewsDSL)
		require.NotNil(t, endpoint.Result)
		require.NotEmpty(t, endpoint.Result.Responses)
		resp := endpoint.Result.Responses[0]
		require.Len(t, resp.ServerBody, 1)
		require.Equal(t, "A", resp.ResultAttr)
		require.Equal(t, "", resp.ServerBody[0].View)
		require.NotNil(t, resp.ClientBody)
	})

	t.Run("explicit method view keeps single projected body", func(t *testing.T) {
		endpoint := firstEndpointData(t, testdata.ResultWithResultViewDSL)
		require.NotNil(t, endpoint.Result)
		require.Equal(t, "full", endpoint.Result.View)
		require.NotEmpty(t, endpoint.Result.Responses)
		resp := endpoint.Result.Responses[0]
		require.Len(t, resp.ServerBody, 1)
		require.NotNil(t, resp.ViewedResult)
		require.Equal(t, "full", resp.ServerBody[0].View)
		require.Equal(t, "", resp.ClientBody.View)
	})

	t.Run("inline object body on multi-view result fans out server bodies per view", func(t *testing.T) {
		endpoint := firstEndpointData(t, testdata.ExplicitBodyUserResultObjectMultipleViewDSL)
		require.NotNil(t, endpoint.Result)
		require.NotEmpty(t, endpoint.Result.Responses)
		resp := endpoint.Result.Responses[0]
		require.Empty(t, resp.ResultAttr)
		require.Len(t, resp.ServerBody, 2)
		require.NotNil(t, resp.ViewedResult)
		require.Equal(t, []string{"default", "tiny"}, bodyViews(resp.ServerBody))
		require.NotNil(t, resp.ClientBody)
		require.Equal(t, "", resp.ClientBody.View)
	})
}

func TestHTTPErrorBodyDescriptionRewrite(t *testing.T) {
	endpoint := firstEndpointData(t, testdata.WithErrorCustomPkgDSL)
	require.NotEmpty(t, endpoint.Errors)
	require.NotEmpty(t, endpoint.Errors[0].Errors)

	errResp := endpoint.Errors[0].Errors[0].Response
	require.NotNil(t, errResp)
	require.NotNil(t, errResp.ClientBody)
	require.Contains(t, errResp.ClientBody.Description, `"ServiceWithErrorCustomPkg" service`)
	require.Contains(t, errResp.ClientBody.Description, `"MethodWithErrorCustomPkg" endpoint`)
	require.Contains(t, errResp.ClientBody.Description, `"error_name" error`)
	require.NotEmpty(t, errResp.ServerBody)
	require.Contains(t, errResp.ServerBody[0].Description, `"ServiceWithErrorCustomPkg" service`)
}

func TestHTTPDirectBuilderSeams(t *testing.T) {
	t.Run("buildEndpointData preserves mixed result assembly", func(t *testing.T) {
		services, endpointExpr, svcData := firstHTTPBuildContext(t, testdata.MixedResultsDSL)

		endpoint := services.buildEndpointData(endpointExpr, svcData.Service, svcData, codegen.NewNameScope())
		require.True(t, endpoint.HasMixedResults)
		require.NotNil(t, endpoint.SSE)
		require.Equal(t, "EncodeCreateRequest", endpoint.RequestEncoder)
		require.Equal(t, "BuildCreateRequest", endpoint.RequestInit.Name)
	})

	t.Run("buildPayloadData projects jsonrpc ids", func(t *testing.T) {
		services, endpointExpr, svcData := firstJSONRPCBuildContext(t, jsonrpcIDProjectionDSL)

		payload := services.buildPayloadData(endpointExpr, svcData)
		require.Equal(t, "ID", payload.IDAttribute)
		require.True(t, payload.IDAttributeRequired)
	})

	t.Run("buildResultData keeps default view and jsonrpc ids", func(t *testing.T) {
		services, endpointExpr, svcData := firstJSONRPCBuildContext(t, jsonrpcIDProjectionDSL)

		result := services.buildResultData(endpointExpr, svcData)
		require.Equal(t, expr.DefaultView, result.View)
		require.Equal(t, "ID", result.IDAttribute)
		require.True(t, result.IDAttributeRequired)
	})

	t.Run("buildRequestBodyType flattens form union helper field", func(t *testing.T) {
		services, endpointExpr, svcData := firstHTTPBuildContext(t, testdata.PayloadFormBodyUnionDSL)

		bodyType := services.buildRequestBodyType(endpointExpr.Body, endpointExpr.MethodExpr.Payload, endpointExpr, false, svcData)
		require.NotNil(t, bodyType)
		require.Equal(t, "Values", bodyType.FlatFormUnionField)
		require.NotNil(t, bodyType.Init)
		require.NotEmpty(t, bodyType.Init.ClientCode)
	})

	t.Run("buildRequestBodyType only emits constructors on the client", func(t *testing.T) {
		services, endpointExpr, svcData := firstHTTPBuildContext(t, testdata.PayloadFormBodyUnionDSL)

		clientBodyType := services.buildRequestBodyType(endpointExpr.Body, endpointExpr.MethodExpr.Payload, endpointExpr, false, svcData)
		serverBodyType := services.buildRequestBodyType(endpointExpr.Body, endpointExpr.MethodExpr.Payload, endpointExpr, true, svcData)
		require.NotNil(t, clientBodyType)
		require.NotNil(t, clientBodyType.Init)
		require.NotNil(t, serverBodyType)
		require.Nil(t, serverBodyType.Init)
	})

	t.Run("buildResponseBodyType keeps projected view names", func(t *testing.T) {
		services, endpointExpr, svcData := firstHTTPBuildContext(t, testdata.ResultWithResultViewDSL)
		method := svcData.Service.Method(endpointExpr.Name())

		bodyType := services.buildResponseBodyType(endpointExpr.Responses[0].Body, endpointExpr.MethodExpr.Result, method.ResultLoc, endpointExpr, true, stringPtr("full"), svcData)
		require.NotNil(t, bodyType)
		require.Equal(t, "full", bodyType.View)
		require.NotNil(t, bodyType.Init)
		require.NotEmpty(t, bodyType.Init.ServerCode)
	})

	t.Run("buildResponseBodyType only emits constructors on the server", func(t *testing.T) {
		services, endpointExpr, svcData := firstHTTPBuildContext(t, testdata.ResultBodyCollectionDSL)
		method := svcData.Service.Method(endpointExpr.Name())

		serverBodyType := services.buildResponseBodyType(endpointExpr.Responses[0].Body, endpointExpr.MethodExpr.Result, method.ResultLoc, endpointExpr, true, nil, svcData)
		clientBodyType := services.buildResponseBodyType(endpointExpr.Responses[0].Body, endpointExpr.MethodExpr.Result, method.ResultLoc, endpointExpr, false, nil, svcData)
		require.NotNil(t, serverBodyType)
		require.NotNil(t, serverBodyType.Init)
		require.NotNil(t, clientBodyType)
		require.Nil(t, clientBodyType.Init)
	})

	t.Run("buildResponseBodyType uses endpoint scoped wrapper for collection bodies", func(t *testing.T) {
		services, endpointExpr, svcData := firstHTTPBuildContext(t, testdata.ResultBodyCollectionDSL)
		method := svcData.Service.Method(endpointExpr.Name())

		bodyType := services.buildResponseBodyType(endpointExpr.Responses[0].Body, endpointExpr.MethodExpr.Result, method.ResultLoc, endpointExpr, true, nil, svcData)
		require.NotNil(t, bodyType)
		require.Equal(t, "MethodBodyCollectionResponseBody", bodyType.Name)
		require.Equal(t, "MethodBodyCollectionResponseBody", bodyType.VarName)
		require.NotNil(t, bodyType.Init)
		require.Equal(t, "NewMethodBodyCollectionResponseBody", bodyType.Init.Name)
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
