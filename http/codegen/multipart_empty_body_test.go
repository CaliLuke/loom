package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	cg "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestMultipartEmptyBodyBindings(t *testing.T) {
	root := RunHTTPDSL(t, testdata.PayloadMultipartWithParamsAndHeadersDSL)
	services := CreateHTTPServices(root)
	request := services.Get("ServiceMultipartWithParamsAndHeaders").Endpoints[0].Payload.Request
	require.Empty(t, request.multipart.Fields)
	dir := t.TempDir()
	renderHTTPModuleFiles(t, dir, "example.com/multipartempty/gen", root)
	for _, role := range []string{"client", "server"} {
		t.Run(role, func(t *testing.T) {
			path := filepath.Join(dir, "gen/http/service_multipart_with_params_and_headers", role, "encode_decode.go")
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			require.NoError(t, err)
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok {
					continue
				}
				// This fixture has no shadowed body/form locals. Count the
				// declarations and uses within each generated function, while
				// the companion generated-module test checks Go's full scopes.
				uses := make(map[string]int)
				locals := make(map[string]bool)
				ast.Inspect(function.Body, func(node ast.Node) bool {
					if id, ok := node.(*ast.Ident); ok {
						uses[id.Name]++
					}
					if assignment, ok := node.(*ast.AssignStmt); ok && assignment.Tok == token.DEFINE {
						for _, lhs := range assignment.Lhs {
							if id, ok := lhs.(*ast.Ident); ok {
								locals[id.Name] = true
							}
						}
					}
					return true
				})
				for _, name := range []string{"body", "multipartForm"} {
					if locals[name] {
						require.Greater(t, uses[name], 1, "%s is declared without a consumer in %s", name, function.Name.Name)
					}
				}
			}
		})
	}
}

func TestMultipartEmptyBodyProjectionModes(t *testing.T) {
	// These render-data variants cover the template's projection contract.
	// Native object bodies have an initializer; the generated-module test
	// below compiles and executes the admitted DSL path.
	for _, mode := range []string{"initializer", "fallible initializer", "direct value"} {
		t.Run(mode, func(t *testing.T) {
			root := RunHTTPDSL(t, testdata.PayloadMultipartWithParamsAndHeadersDSL)
			services := CreateHTTPServices(root)
			svc := root.API.HTTP.Services[0]
			endpoint := services.Get(svc.Name()).Endpoints[0]
			request := endpoint.Payload.Request
			switch mode {
			case "fallible initializer":
				request.ClientBody.Init.ClientErrorAware = true
			case "direct value":
				request.ClientBody.Init = nil
				request.ClientBodyValue = requestClientBodyValue(request)
			}
			code := cg.SectionCode(t, clientRequestEncoderSection(svc, services, endpoint))
			require.NotContains(t, code, "body :=")
			if mode == "direct value" {
				require.NotContains(t, code, "_ = p")
				return
			}
			require.Equal(t, 1, strings.Count(code, request.ClientBodyValue), "evaluate the projection once")
			if mode == "fallible initializer" {
				require.Contains(t, code, "_, conversionError := "+request.ClientBodyValue)
				require.Contains(t, code, "if conversionError != nil")
				require.Contains(t, code, `return loomhttp.ErrEncodingError("ServiceMultipartWithParamsAndHeaders", "MethodMultipartWithParamsAndHeaders", conversionError)`)
			} else {
				require.NotContains(t, code, "_ = "+request.ClientBodyValue)
			}
		})
	}
}

func TestMultipartClosedEmptyBodyKeepsParserValue(t *testing.T) {
	root := RunHTTPDSL(t, testdata.PayloadMultipartWithParamsAndHeadersDSL)
	request := CreateHTTPServices(root).Get("ServiceMultipartWithParamsAndHeaders").Endpoints[0].Payload.Request
	request.multipart.ObjectOpenness = expr.ObjectClosed
	code := renderMultipartRequestDecode(request)
	require.Contains(t, code, "multipartForm, multipartErr := loomhttp.ReadMultipartForm(r)")
	require.Contains(t, code, "loomhttp.MultipartObjectJSON(multipartForm,")
	require.Contains(t, code, "undeclared multipart part")
}

func TestMultipartEmptyBodyGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, testdata.PayloadMultipartWithParamsAndHeadersDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/multipartempty", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "multipart_test.go"), []byte(multipartEmptyBodyHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
	runGoCommand(t, dir, "vet", "./...")
}

const multipartEmptyBodyHarness = `package multipartempty_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	client "example.com/multipartempty/gen/http/service_multipart_with_params_and_headers/client"
	server "example.com/multipartempty/gen/http/service_multipart_with_params_and_headers/server"
	svc "example.com/multipartempty/gen/service_multipart_with_params_and_headers"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestEmptyMultipartBoundInputs(t *testing.T) {
	for _, tc := range []struct {
		name, path, query, header, source, problem, detail string
		status                                             int
	}{
		{name: "empty envelope retains all bindings", path: "patterna", query: "c=" + url.QueryEscape("{\"1\":[\"x\"]}"), header: "patternb", status: 200},
		{name: "open body ignores unknown part", path: "patterna", query: "c=" + url.QueryEscape("{\"1\":[\"x\"]}"), header: "patternb", source: "extra", status: 200},
		{name: "required query still validated", path: "patterna", header: "patternb", status: 400, problem: "missing_field", detail: "Missing required field: c"},
		{name: "path still validated", path: "bad", query: "c=" + url.QueryEscape("{\"1\":[\"x\"]}"), header: "patternb", status: 400, problem: "invalid_pattern", detail: "validation error"},
		{name: "header still validated", path: "patterna", query: "c=" + url.QueryEscape("{\"1\":[\"x\"]}"), header: "bad", status: 400, problem: "invalid_pattern", detail: "validation error"},
		{name: "required body absence", path: "patterna", query: "c=" + url.QueryEscape("{\"1\":[\"x\"]}"), source: "absent", status: 400, problem: "missing_payload", detail: "validation error"},
		{name: "malformed envelope", path: "patterna", query: "c=" + url.QueryEscape("{\"1\":[\"x\"]}"), source: "malformed", status: 400, problem: "decode_payload", detail: "invalid request body"},
		{name: "source read failure", path: "patterna", query: "c=" + url.QueryEscape("{\"1\":[\"x\"]}"), source: "read failure", status: 400, problem: "decode_payload", detail: "invalid request body"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buffer bytes.Buffer
			writer := multipart.NewWriter(&buffer)
			if tc.source == "extra" {
				require.NoError(t, writer.WriteField("extra", "ignored"))
			}
			require.NoError(t, writer.Close())
			request := httptest.NewRequest("POST", "/"+tc.path+"?"+tc.query, &buffer)
			request.Header.Set("Content-Type", writer.FormDataContentType())
			if tc.source == "absent" {
				request.Body = http.NoBody
				request.ContentLength = 0
			}
			if tc.source == "malformed" {
				request.Body = io.NopCloser(strings.NewReader("broken"))
				request.ContentLength = 6
			}
			if tc.source == "read failure" {
				request.Body = failedMultipartSource{}
				request.ContentLength = -1
			}
			if tc.header != "" {
				request.Header.Set("Authorization", tc.header)
			}
			calls := 0
			endpoint := func(_ context.Context, raw any) (any, error) {
				calls++
				payload, ok := raw.(*svc.PayloadType)
				require.True(t, ok)
				require.NotNil(t, payload)
				require.Equal(t, "patterna", payload.A)
				require.NotNil(t, payload.B)
				require.Equal(t, "patternb", *payload.B)
				require.Equal(t, map[int][]string{1: {"x"}}, payload.C)
				return "ok", nil
			}
			mux := loomhttp.NewMuxer()
			server.Mount(mux, server.New(&svc.Endpoints{MethodMultipartWithParamsAndHeaders: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			require.Equal(t, tc.status, response.Code, response.Body.String())
			if tc.status == 200 {
				require.Equal(t, 1, calls)
			} else {
				require.Zero(t, calls)
				var problem map[string]any
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
				require.Equal(t, tc.problem, problem["code"])
				require.Equal(t, tc.detail, problem["detail"])
			}
		})
	}
}

func TestEmptyMultipartClientKeepsEnvelopeAndBindings(t *testing.T) {
	header := "patternb"
	payload := &svc.PayloadType{A: "patterna", B: &header, C: map[int][]string{1: {"x"}}}
	request := httptest.NewRequest("POST", "/patterna", nil)
	require.NoError(t, client.EncodeMethodMultipartWithParamsAndHeadersRequest(loomhttp.RequestEncoder)(request, payload))
	require.Equal(t, "patternb", request.Header.Get("Authorization"))
	require.JSONEq(t, "{\"1\":[\"x\"]}", request.URL.Query().Get("c"))
	form, err := loomhttp.ReadMultipartForm(request)
	require.NoError(t, err)
	require.Empty(t, form.Parts)
}

type failedMultipartSource struct{}

func (failedMultipartSource) Read([]byte) (int, error) {
	return 0, errors.New("source read failed")
}
func (failedMultipartSource) Close() error {
	return nil
}
`
