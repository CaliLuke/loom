package service

import (
	"bytes"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
)

func TestClientJenniferRequestExpr(t *testing.T) {
	t.Run("skip request body wraps payload and body", func(t *testing.T) {
		method := &EndpointMethodData{
			MethodData: &MethodData{
				PayloadRef:                  "*Payload",
				SkipRequestBodyEncodeDecode: true,
				RequestStruct:               "MethodRequestData",
			},
		}
		rendered := renderJenniferExpr(t, requestExpr(method))
		assert.Contains(t, rendered, "&MethodRequestData{Payload: p, Body: req}")
	})

	t.Run("payload only uses payload value", func(t *testing.T) {
		method := &EndpointMethodData{
			MethodData: &MethodData{PayloadRef: "*Payload"},
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
				VarName:            "Method",
				ClientInterceptors: []string{"logging"},
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
				Name:                        "Upload",
				VarName:                     "Upload",
				PayloadRef:                  "*UploadPayload",
				SkipRequestBodyEncodeDecode: true,
				RequestStruct:               "UploadRequestData",
				EndpointField:               "UploadEndpoint",
			},
		}
		code := codegen.SectionCode(t, methodSection(method))
		assert.Contains(t, code, "func (c *Client) Upload(ctx context.Context, p *UploadPayload, req io.ReadCloser)")
		assert.Contains(t, code, "c.UploadEndpoint(ctx, &UploadRequestData{Payload: p, Body: req})")
	})

	t.Run("skip response body decode unwraps response struct", func(t *testing.T) {
		method := &EndpointMethodData{
			ClientVarName: "Client",
			ServiceName:   "DownloadService",
			MethodData: &MethodData{
				Name:                         "Download",
				VarName:                      "Download",
				ResultRef:                    "*DownloadResult",
				SkipResponseBodyEncodeDecode: true,
				ResponseStruct:               "DownloadResponseData",
				EndpointField:                "DownloadEndpoint",
			},
		}
		code := codegen.SectionCode(t, methodSection(method))
		assert.Contains(t, code, "func (c *Client) Download(ctx context.Context) (res *DownloadResult, resp io.ReadCloser, err error)")
		assert.Contains(t, code, "o := ires.(*DownloadResponseData)")
		assert.Contains(t, code, "return o.Result, o.Body, nil")
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
					Name:                "Method",
					VarName:             "Method",
					EndpointField:       "MethodEndpoint",
					StreamEndpointField: "MethodStreamEndpoint",
					HasMixedResults:     true,
					ClientInterceptors:  []string{"logging"},
				},
			}},
		}
		code := codegen.SectionCode(t, clientInitSection(data))
		assert.Contains(t, code, "MethodEndpoint: WrapMethodClientEndpoint(method, ci)")
		assert.Contains(t, code, "MethodStreamEndpoint: WrapMethodClientEndpoint(methodStream, ci)")
	})

	t.Run("method comments and error comments remain valid go", func(t *testing.T) {
		method := &EndpointMethodData{
			ClientVarName: "Client",
			ServiceName:   "AccountService",
			MethodData: &MethodData{
				Name:          "ListAccounts",
				VarName:       "ListAccounts",
				Description:   "ListAccounts retrieves accounts.",
				EndpointField: "ListAccountsEndpoint",
				Errors: []*ErrorInitData{{
					ErrName:     "quota_exceeded",
					TypeRef:     "*QuotaExceededError",
					Description: "Returned when quota is exhausted.",
				}},
			},
		}
		code := codegen.SectionCode(t, methodSection(method))
		require.Contains(t, code, `// ListAccounts calls the "ListAccounts" endpoint of the "AccountService"`)
		require.Contains(t, code, `// - "quota_exceeded" (type *QuotaExceededError): Returned when quota is`)
		require.Contains(t, code, `// exhausted.`)
		require.Contains(t, code, "// - error: internal error")
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
