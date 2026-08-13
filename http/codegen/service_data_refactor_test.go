package codegen

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/testdata"
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

func TestHTTPOptionalRequestBodyAssembly(t *testing.T) {
	cases := []struct {
		name            string
		dsl             func()
		wantBodyOrigin  string
		wantPayloadAttr string
	}{
		{
			name: "whole payload body",
			dsl:  testdata.PayloadBodyObjectOptionalRequestDSL,
		},
		{
			name:            "optional payload attribute body",
			dsl:             testdata.PayloadBodyObjectOptionalOriginRequestDSL,
			wantBodyOrigin:  "body",
			wantPayloadAttr: "Body",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			services, endpointExpr, svcData := firstHTTPBuildContext(t, c.dsl)
			endpointIR := transportir.BuildEndpoint(endpointExpr)
			payload := services.buildPayloadDataFromIR(endpointIR, svcData)

			require.True(t, endpointIR.Request.OptionalBody)
			require.False(t, endpointIR.Request.MustHaveBody)
			require.Equal(t, c.wantBodyOrigin, endpointIR.Request.BodyOrigin)
			require.NotNil(t, payload.Request)
			require.False(t, payload.Request.MustHaveBody)
			require.False(t, payload.Request.MustValidate)
			require.Equal(t, c.wantPayloadAttr, payload.Request.PayloadAttr)
			require.NotNil(t, payload.Request.DecodePlan)
			require.False(t, payload.Request.DecodePlan.MustValidate)
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
		expectErrorVar     bool
	}{
		{
			name:               "custom multipart decoder required",
			dsl:                testdata.PayloadMultipartPrimitiveDSL,
			expectDecoder:      true,
			expectGenerated:    false,
			expectFileFieldLen: 0,
			expectErrorVar:     true,
		},
		{
			name:               "generated multipart object",
			dsl:                testdata.PayloadMultipartObjectGeneratedDSL,
			expectDecoder:      false,
			expectGenerated:    true,
			expectFileFieldLen: 1,
			expectErrorVar:     true,
		},
		{
			name:               "generated optional multipart object",
			dsl:                testdata.PayloadMultipartObjectGeneratedOptionalDSL,
			expectDecoder:      false,
			expectGenerated:    true,
			expectFileFieldLen: 1,
			expectErrorVar:     false,
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
			require.Equal(t, c.expectErrorVar, endpoint.Payload.Request.NeedsServerErrorVar)
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
			name:                 "documented raw request body",
			dsl:                  testdata.RawRequestBodyOpenAPIDSL,
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

func TestDocumentedRawRequestBodyPreservesRawTransportGeneration(t *testing.T) {
	endpoint := firstEndpointData(t, testdata.RawRequestBodyOpenAPIDSL)

	require.True(t, endpoint.Method.SkipRequestBodyEncodeDecode)
	require.Nil(t, endpoint.Payload.Request.ServerBody)
	require.Nil(t, endpoint.Payload.Request.ClientBody)
	require.NotEmpty(t, endpoint.BuildStreamPayload)
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

func TestHTTPErrorResponseContentTypeDefaulting(t *testing.T) {
	cases := []struct {
		name string
		dsl  func()
		want string
	}{
		{
			name: "default error response uses problem json",
			dsl:  testdata.DefaultErrorResponseDSL,
			want: "application/problem+json",
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

func TestBuildProblemClientResultTransformCodeWithoutBody(t *testing.T) {
	args := []*InitArgData{
		{AttributeData: &AttributeData{Name: "code", VarName: "code"}},
	}

	got := buildProblemClientResultTransformCode(&transportir.ResponseStatus{StatusCode: http.StatusInternalServerError}, false, args)

	require.Equal(t, `v := loomhttp.ProblemErrorFromBody(code, 500, "", "", nil)`, got)
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
		require.Equal(t, "full", resp.ClientBody.View)
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

func TestHTTPProjectionParity(t *testing.T) {
	endpoint := firstEndpointData(t, testdata.ExplicitBodyUserResultObjectMultipleViewDSL)
	require.NotNil(t, endpoint.Method.ViewedResult)
	require.NotNil(t, endpoint.Result)
	require.NotEmpty(t, endpoint.Result.Responses)

	resp := endpoint.Result.Responses[0]
	require.NotNil(t, resp.ViewedResult)
	require.Equal(t, endpoint.Method.ViewedResult.FullRef, resp.ViewedResult.FullRef)
	require.Equal(t, []string{"default", "tiny"}, bodyViews(resp.ServerBody))
	require.Equal(t, []string{"default", "tiny"}, viewNames(resp.ViewedResult.Views))
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

func viewNames(views []*service.ViewData) []string {
	names := make([]string, len(views))
	for i, view := range views {
		names[i] = view.Name
	}
	return names
}
