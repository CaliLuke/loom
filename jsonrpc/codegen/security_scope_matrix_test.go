package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr/testdata"
)

func TestJSONRPCSecurityScopeMatrix(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.SecurityScopeMatrixDSL("jsonrpc"))
	services := CreateJSONRPCServices(root)

	for _, tc := range []struct {
		service string
		method  string
		want    []string
	}{
		{service: "api_scope", method: "read", want: []string{"basic"}},
		{service: "scope", method: "inherited", want: []string{"jwt"}},
		{service: "scope", method: "override", want: []string{"key"}},
		{service: "scope", method: "alternatives", want: []string{"basic", "jwt"}},
		{service: "scope", method: "open"},
	} {
		t.Run(tc.service+"/"+tc.method, func(t *testing.T) {
			endpoint := services.Get(tc.service).Endpoint(tc.method)
			require.NotNil(t, endpoint)
			require.Len(t, endpoint.Method.Requirements, len(tc.want))
			for _, name := range tc.want {
				require.NotNil(t, endpoint.Method.Requirements.Scheme(name), name)
			}
		})
	}

	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/securityscope", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scope_test.go"), []byte(jsonrpcSecurityScopeMatrixHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}

const jsonrpcSecurityScopeMatrixHarness = `package securityscope_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	apisvc "example.com/securityscope/gen/api_scope"
	apisvr "example.com/securityscope/gen/jsonrpc/api_scope/server"
	svc "example.com/securityscope/gen/scope"
	svr "example.com/securityscope/gen/jsonrpc/scope/server"
	loom "github.com/CaliLuke/loom/pkg"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/security"
)

type implementation struct {
	calls map[string]int
	auth  []string
}

func (s *implementation) BasicAuth(ctx context.Context, user, pass string, _ *security.BasicScheme) (context.Context, error) {
	s.auth = append(s.auth, "basic")
	if user == "user" && pass == "pass" {
		return ctx, nil
	}
	return ctx, loom.PermanentError("unauthorized", "invalid credentials")
}

func (s *implementation) JWTAuth(ctx context.Context, token string, _ *security.JWTScheme) (context.Context, error) {
	s.auth = append(s.auth, "jwt")
	if token == "jwt-ok" {
		return ctx, nil
	}
	return ctx, loom.PermanentError("unauthorized", "invalid credentials")
}

func (s *implementation) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	s.auth = append(s.auth, "key")
	if key == "key-ok" {
		return ctx, nil
	}
	return ctx, loom.PermanentError("unauthorized", "invalid credentials")
}

func (s *implementation) Read(context.Context, *apisvc.ReadPayload) error {
	s.calls["api_scope.read"]++
	return nil
}

func (s *implementation) Inherited(context.Context, *svc.InheritedPayload) error {
	s.calls["scope.inherited"]++
	return nil
}

func (s *implementation) Override(context.Context, *svc.OverridePayload) error {
	s.calls["scope.override"]++
	return nil
}

func (s *implementation) Alternatives(context.Context, *svc.AlternativesPayload) error {
	s.calls["scope.alternatives"]++
	return nil
}

func (s *implementation) Open(context.Context, *svc.OpenPayload) error {
	s.calls["scope.open"]++
	return nil
}

func TestSecurityScopeMatrixAdapter(t *testing.T) {
	impl := &implementation{calls: map[string]int{}}
	mux := loomhttp.NewMuxer()
	apisvr.Mount(mux, apisvr.New(apisvc.NewEndpoints(impl), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	svr.Mount(mux, svr.New(svc.NewEndpoints(impl), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))

	for _, tc := range []struct {
		name, service, method, authorization string
		wantAuth                            []string
		wantCall                            bool
	}{
		{name: "api inherited accepted", service: "api_scope", method: "read", authorization: "Basic dXNlcjpwYXNz", wantAuth: []string{"basic"}, wantCall: true},
		{name: "api inherited rejected", service: "api_scope", method: "read", authorization: "Basic dXNlcjp3cm9uZw==", wantAuth: []string{"basic"}},
		{name: "service inherited accepted", service: "scope", method: "inherited", authorization: "Bearer jwt-ok", wantAuth: []string{"jwt"}, wantCall: true},
		{name: "service inherited rejected", service: "scope", method: "inherited", authorization: "Bearer jwt-bad", wantAuth: []string{"jwt"}},
		{name: "method override accepted", service: "scope", method: "override", authorization: "key-ok", wantAuth: []string{"key"}, wantCall: true},
		{name: "method override rejected", service: "scope", method: "override", authorization: "key-bad", wantAuth: []string{"key"}},
		{name: "alternative basic accepted", service: "scope", method: "alternatives", authorization: "Basic dXNlcjpwYXNz", wantAuth: []string{"basic"}, wantCall: true},
		{name: "alternative jwt accepted", service: "scope", method: "alternatives", authorization: "Bearer jwt-ok", wantAuth: []string{"basic", "jwt"}, wantCall: true},
		{name: "alternative rejected", service: "scope", method: "alternatives", authorization: "Bearer jwt-bad", wantAuth: []string{"basic", "jwt"}},
		{name: "no security without credentials", service: "scope", method: "open", wantCall: true},
		{name: "no security ignores credentials", service: "scope", method: "open", authorization: "Bearer jwt-bad", wantCall: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			beforeCalls := cloneCalls(impl.calls)
			beforeAuth := len(impl.auth)
			params := map[string]any{"message": "hello"}
			body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": "request", "method": tc.method, "params": params})
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "/"+tc.service, strings.NewReader(string(body)))
			req.Header.Set("Content-Type", "application/json")
			if tc.authorization != "" {
				req.Header.Set("Authorization", tc.authorization)
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			var response map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response), rec.Body.String())
			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, "2.0", response["jsonrpc"], rec.Body.String())
			require.Equal(t, "request", response["id"], rec.Body.String())
			if tc.wantCall {
				require.Contains(t, response, "result", rec.Body.String())
				require.NotContains(t, response, "error", rec.Body.String())
			} else {
				require.NotContains(t, response, "result", rec.Body.String())
				failure, ok := response["error"].(map[string]any)
				require.True(t, ok, rec.Body.String())
				require.Equal(t, float64(-32603), failure["code"], rec.Body.String())
				require.Equal(t, "invalid credentials", failure["message"], rec.Body.String())
				data, ok := failure["data"].(map[string]any)
				require.True(t, ok, rec.Body.String())
				require.Equal(t, "unauthorized", data["name"], rec.Body.String())
				require.NotEmpty(t, data["id"], rec.Body.String())
				require.Equal(t, false, data["temporary"], rec.Body.String())
				require.Equal(t, false, data["timeout"], rec.Body.String())
				require.Equal(t, false, data["fault"], rec.Body.String())
			}
			serviceMethod := tc.service + "." + tc.method
			wantServiceCalls := 0
			if tc.wantCall {
				wantServiceCalls = 1
			}
			require.Equal(t, wantServiceCalls, impl.calls[serviceMethod]-beforeCalls[serviceMethod])
			for calledMethod, calls := range impl.calls {
				if calledMethod == serviceMethod {
					continue
				}
				require.Equal(t, beforeCalls[calledMethod], calls, calledMethod)
			}
			if len(tc.wantAuth) == 0 {
				require.Empty(t, impl.auth[beforeAuth:])
			} else {
				require.Equal(t, tc.wantAuth, impl.auth[beforeAuth:])
			}
		})
	}
}

func cloneCalls(calls map[string]int) map[string]int {
	result := make(map[string]int, len(calls))
	for key, value := range calls {
		result[key] = value
	}
	return result
}

`
