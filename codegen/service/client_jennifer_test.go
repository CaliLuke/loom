package service

import (
	"bytes"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
)

func TestClientJenniferRequestExpr(t *testing.T) {
	t.Run("skip request body wraps payload and body", func(t *testing.T) {
		method := &EndpointMethodData{
			MethodData: &MethodData{
				MethodPayloadData: MethodPayloadData{
					PayloadRef: "*Payload",
				},
				MethodTransportData: MethodTransportData{
					SkipRequestBodyEncodeDecode: true,
					RequestStruct:               "MethodRequestData",
				},
			},
		}
		rendered := renderJenniferExpr(t, requestExpr(method))
		assert.Contains(t, rendered, "&MethodRequestData{Payload: p, Body: req}")
	})

	t.Run("payload only uses payload value", func(t *testing.T) {
		method := &EndpointMethodData{
			MethodData: &MethodData{
				MethodPayloadData: MethodPayloadData{PayloadRef: "*Payload"},
			},
		}
		rendered := renderJenniferExpr(t, requestExpr(method))
		assert.Contains(t, rendered, "var _ = p")
	})

	t.Run("empty request uses nil", func(t *testing.T) {
		method := &EndpointMethodData{MethodData: &MethodData{}}
		rendered := renderJenniferExpr(t, requestExpr(method))
		assert.Contains(t, rendered, "var _ = nil")
	})
}

func TestClientJenniferWrappedEndpointExpr(t *testing.T) {
	t.Run("plain endpoint uses arg name", func(t *testing.T) {
		method := &EndpointMethodData{ArgName: "endpoint", MethodData: &MethodData{VarName: "Method"}}
		rendered := renderJenniferExpr(t, wrappedClientEndpointExpr(method, false))
		assert.Contains(t, rendered, "var _ = endpoint")
	})

	t.Run("intercepted mixed stream endpoint uses stream arg", func(t *testing.T) {
		method := &EndpointMethodData{
			ArgName:       "endpoint",
			StreamArgName: "streamEndpoint",
			MethodData: &MethodData{
				VarName: "Method",
				MethodSecurityData: MethodSecurityData{
					ClientInterceptors: []string{"logging"},
				},
			},
		}
		rendered := renderJenniferExpr(t, wrappedClientEndpointExpr(method, true))
		assert.Contains(t, rendered, "WrapMethodClientEndpoint(streamEndpoint, ci)")
	})
}

func TestClientJenniferMethodSectionVariants(t *testing.T) {
	t.Run("skip request body decode passes request body through", func(t *testing.T) {
		method := &EndpointMethodData{
			ClientVarName: "Client",
			ServiceName:   "UploadService",
			MethodData: &MethodData{
				Name:    "Upload",
				VarName: "Upload",
				MethodPayloadData: MethodPayloadData{
					PayloadRef: "*UploadPayload",
				},
				MethodTransportData: MethodTransportData{
					SkipRequestBodyEncodeDecode: true,
					RequestStruct:               "UploadRequestData",
					EndpointField:               "UploadEndpoint",
				},
			},
		}
		code := codegen.SectionCode(t, methodSection(method))
		assert.Contains(t, code, "func (c *Client) Upload(ctx context.Context, p *UploadPayload, req io.ReadCloser)")
		testutil.AssertGo(t, "testdata/golden/client_method_skip_request_body.go.golden", code)
	})

	t.Run("skip response body decode unwraps response struct", func(t *testing.T) {
		method := &EndpointMethodData{
			ClientVarName: "Client",
			ServiceName:   "DownloadService",
			MethodData: &MethodData{
				Name:    "Download",
				VarName: "Download",
				MethodResultData: MethodResultData{
					ResultRef: "*DownloadResult",
				},
				MethodTransportData: MethodTransportData{
					SkipResponseBodyEncodeDecode: true,
					ResponseStruct:               "DownloadResponseData",
					EndpointField:                "DownloadEndpoint",
				},
			},
		}
		code := codegen.SectionCode(t, methodSection(method))
		assert.Contains(t, code, "func (c *Client) Download(ctx context.Context) (res *DownloadResult, resp io.ReadCloser, err error)")
		testutil.AssertGo(t, "testdata/golden/client_method_skip_response_body.go.golden", code)
	})

	t.Run("mixed results with interceptors uses both endpoint fields", func(t *testing.T) {
		data := &EndpointsData{
			Name:                  "MixedInterceptorService",
			ClientVarName:         "Client",
			ClientInitArgs:        "method, methodStream",
			HasClientInterceptors: true,
			Methods: []*EndpointMethodData{{
				ArgName:       "method",
				StreamArgName: "methodStream",
				MethodData: &MethodData{
					Name:    "Method",
					VarName: "Method",
					MethodSecurityData: MethodSecurityData{
						ClientInterceptors: []string{"logging"},
					},
					MethodTransportData: MethodTransportData{
						EndpointField:       "MethodEndpoint",
						StreamEndpointField: "MethodStreamEndpoint",
					},
					MethodStreamingData: MethodStreamingData{
						HasMixedResults: true,
					},
				},
			}},
		}
		code := codegen.SectionCode(t, clientInitSection(data))
		assert.Contains(t, code, "MethodStreamEndpoint: WrapMethodClientEndpoint(methodStream, ci)")
		testutil.AssertGo(t, "testdata/golden/client_init_mixed_interceptors.go.golden", code)
	})

	t.Run("method comments and error comments remain valid go", func(t *testing.T) {
		method := &EndpointMethodData{
			ClientVarName: "Client",
			ServiceName:   "AccountService",
			MethodData: &MethodData{
				Name:        "ListAccounts",
				VarName:     "ListAccounts",
				Description: "ListAccounts retrieves accounts.",
				MethodSecurityData: MethodSecurityData{
					Errors: []*ErrorInitData{{
						ErrName:     "quota_exceeded",
						TypeRef:     "*QuotaExceededError",
						Description: "Returned when quota is exhausted.",
					}},
				},
				MethodTransportData: MethodTransportData{
					EndpointField: "ListAccountsEndpoint",
				},
			},
		}
		code := codegen.SectionCode(t, methodSection(method))
		require.Contains(t, code, `// - "quota_exceeded" (type *QuotaExceededError): Returned when quota is`)
		require.Contains(t, code, `// exhausted.`)
		testutil.AssertGo(t, "testdata/golden/client_method_error_comments.go.golden", code)
	})
}

func renderJenniferExpr(t *testing.T, expr *jen.Statement) string {
	t.Helper()

	var buf bytes.Buffer
	stmt := jen.Empty()
	stmt.Var().Id("_").Op("=").Add(expr)
	require.NoError(t, stmt.Render(&buf))

	return codegen.FormatTestCode(t, "package foo\n"+buf.String())
}
