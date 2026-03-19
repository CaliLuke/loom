package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "goa.design/goa/v3/dsl"
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

func lastQueryParam(t *testing.T, endpoint *EndpointData) *ParamData {
	t.Helper()

	require.NotEmpty(t, endpoint.Payload.Request.QueryParams)
	return endpoint.Payload.Request.QueryParams[len(endpoint.Payload.Request.QueryParams)-1]
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
