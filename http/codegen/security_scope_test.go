package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr/testdata"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
)

func TestHTTPSecurityScope(t *testing.T) {
	for _, session := range []bool{false, true} {
		t.Run(map[bool]string{false: "basic", true: "session"}[session], func(t *testing.T) {
			root := RunHTTPDSL(t, testdata.SecurityScopeDSL("http", session))
			spec := renderOpenAPIJSON(t, openapiv3.Files, root)
			parseOpenAPIV3Document(t, spec)
			security := operationFromSpec(t, spec, "/", "get")["security"]
			if session {
				require.Equal(t, []any{map[string]any{"jwt": []any{}}, map[string]any{"key": []any{}}}, security)
			} else {
				require.Equal(t, []any{map[string]any{"basic": []any{}}}, security)
			}
			endpoint := CreateHTTPServices(root).Get("scope").Endpoint("read")
			if session {
				require.Nil(t, endpoint.Method.Requirements.Scheme("basic"))
				require.Len(t, endpoint.Method.Requirements, 2)
			} else {
				require.Len(t, endpoint.Method.Requirements, 1)
				require.NotNil(t, endpoint.Method.Requirements.Scheme("basic"))
				require.Empty(t, endpoint.Payload.Request.Cookies)
				require.NotContains(t, endpoint.Method.PayloadDef, "Auth *string")
			}
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/scope", root)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "endpoint_test.go"), []byte(testdata.SecurityScopeEndpointTest(session)), 0o600))
			if !session {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "scope_test.go"), []byte(httpSecurityScopeHarness), 0o600))
			}
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "test", "./...")
		})
	}
}

const httpSecurityScopeHarness = `package scope_test

import (
 "context"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 svc "example.com/scope/gen/scope"
 server "example.com/scope/gen/http/scope/server"
 loom "github.com/CaliLuke/loom/pkg"
 loomhttp "github.com/CaliLuke/loom/http"
 "github.com/CaliLuke/loom/security"
)

type implementation struct {
 svc.Service
 calls int
 basic int
}
func (s *implementation) BasicAuth(ctx context.Context, user, pass string, _ *security.BasicScheme) (context.Context, error) {
 s.basic++
 if user == "user" && pass == "pass" {
  return ctx, nil
 }
 return ctx, loom.PermanentError("unauthorized", "invalid credentials")
}
func (s *implementation) Read(context.Context, *svc.ReadPayload) error {
 s.calls++
 return nil
}
func TestBasicScope(t *testing.T) {
 impl := &implementation{}
 mux := loomhttp.NewMuxer()
 server.Mount(mux, server.New(svc.NewEndpoints(impl), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
 for _, tc := range []struct { name, header string; cookie, want bool }{
  {"basic", "Basic dXNlcjpwYXNz", false, true},
  {"bearer", "Bearer session-token", false, false},
  {"cookie", "", true, false},
  {"absent", "", false, false},
 } {
  t.Run(tc.name, func(t *testing.T) {
   before := impl.calls
   req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader("{}"))
   req.Header.Set("Authorization", tc.header)
   req.Header.Set("Content-Type", "application/json")
   if tc.cookie {
    req.AddCookie(&http.Cookie{Name:"session", Value:"session-token"})
   }
   rec := httptest.NewRecorder()
   mux.ServeHTTP(rec, req)
   if (impl.calls > before) != tc.want {
    t.Errorf("status %d, service called %v", rec.Code, impl.calls > before)
   }
   if tc.want && rec.Code != http.StatusNoContent {
    t.Errorf("successful status %d", rec.Code)
   }
  })
 }
 if impl.basic != 4 {
  t.Errorf("Basic callback called %d times", impl.basic)
 }
}
`
