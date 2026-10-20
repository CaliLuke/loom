package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	servicecodegen "github.com/CaliLuke/loom/codegen/service"
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
			} else {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "scope_session_test.go"), []byte(httpSessionSecurityScopeHarness), 0o600))
			}
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "test", "./...")
		})
	}
}

func TestHTTPSecurityScopeMatrixProjection(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SecurityScopeMatrixDSL("http"))
	apiServices := CreateHTTPServices(root)
	apiEndpoint := apiServices.Get("api_scope").Endpoint("read")
	require.Len(t, apiEndpoint.Method.Requirements, 1)
	require.NotNil(t, apiEndpoint.Method.Requirements.Scheme("basic"))

	services := CreateHTTPServices(root)
	for _, tc := range []struct {
		method string
		want   []string
	}{
		{method: "inherited", want: []string{"jwt"}},
		{method: "override", want: []string{"key"}},
		{method: "alternatives", want: []string{"basic", "jwt"}},
		{method: "open"},
	} {
		endpoint := services.Get("scope").Endpoint(tc.method)
		require.NotNil(t, endpoint)
		require.Equal(t, tc.want, requirementSchemeNames(endpoint.Method.Requirements), tc.method)
	}
}

func TestHTTPSecurityScopeMatrixRuntime(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SecurityScopeMatrixDSL("http"))
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/scope", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scope_matrix_test.go"), []byte(httpSecurityScopeMatrixHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func requirementSchemeNames(requirements servicecodegen.RequirementsData) []string {
	var names []string
	for _, requirement := range requirements {
		for _, scheme := range requirement.Schemes {
			names = append(names, scheme.SchemeName)
		}
	}
	return names
}

const httpSecurityScopeHarness = `package scope_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	server "example.com/scope/gen/http/scope/server"
	svc "example.com/scope/gen/scope"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
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
	for _, tc := range []struct {
		name, header string
		want         bool
	}{
		{"basic", "Basic dXNlcjpwYXNz", true},
		{"bad basic", "Basic dXNlcjpiYWQ=", false},
		{"bearer", "Bearer session-token", false},
		{"absent", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := impl.calls
			req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader("{}"))
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if (impl.calls > before) != tc.want {
				t.Errorf("status %d, service called %v", rec.Code, impl.calls > before)
			}
			if tc.want && rec.Code != http.StatusNoContent {
				t.Errorf("successful status %d", rec.Code)
			}
			if !tc.want {
				assertUnauthorizedProblem(t, rec)
			}
		})
	}
	if impl.basic != 4 {
		t.Errorf("Basic callback called %d times", impl.basic)
	}
}

func assertUnauthorizedProblem(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("rejected status %d, want %d: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	var problem loomhttp.ProblemResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Errorf("decode problem response %q: %v", rec.Body.String(), err)
		return
	}
	if problem.Code != "unauthorized" {
		t.Errorf("problem code %q, want unauthorized", problem.Code)
	}
	if problem.Detail != "invalid credentials" {
		t.Errorf("problem detail %q, want %q", problem.Detail, "invalid credentials")
	}
}
`

const httpSessionSecurityScopeHarness = `package scope_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	server "example.com/scope/gen/http/scope/server"
	svc "example.com/scope/gen/scope"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type sessionImplementation struct {
	svc.Service
	calls, jwt, key int
}

func (s *sessionImplementation) JWTAuth(ctx context.Context, token string, _ *security.JWTScheme) (context.Context, error) {
	s.jwt++
	if token == "session-token" {
		return ctx, nil
	}
	return ctx, loom.PermanentError("unauthorized", "invalid credentials")
}
func (s *sessionImplementation) APIKeyAuth(ctx context.Context, token string, _ *security.APIKeyScheme) (context.Context, error) {
	s.key++
	if token == "session-token" {
		return ctx, nil
	}
	return ctx, loom.PermanentError("unauthorized", "invalid credentials")
}
func (s *sessionImplementation) Read(context.Context, *svc.ReadPayload) error {
	s.calls++
	return nil
}

func TestHTTPSelectedSessionScope(t *testing.T) {
	impl := &sessionImplementation{}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(svc.NewEndpoints(impl), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	for _, tc := range []struct {
		name, header, cookie, problemCode, problemDetail string
		want, presentEmptyHeader                         bool
		wantJWT, wantKey                                 int
	}{
		{name: "bearer", header: "Bearer session-token", want: true, wantJWT: 1},
		{name: "bad bearer", header: "Bearer wrong", problemCode: "unauthorized", problemDetail: "invalid credentials", wantJWT: 1, wantKey: 1},
		{name: "cookie", cookie: "session-token", want: true, wantJWT: 1, wantKey: 1},
		{name: "bad cookie", cookie: "wrong", problemCode: "unauthorized", problemDetail: "invalid credentials", wantJWT: 1, wantKey: 1},
		{name: "absent", problemCode: "unauthorized", problemDetail: "invalid credentials", wantJWT: 1, wantKey: 1},
		{name: "empty bearer header", presentEmptyHeader: true, problemCode: "decode_payload", problemDetail: "invalid Authorization bearer credential"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := impl.calls
			beforeJWT, beforeKey := impl.jwt, impl.key
			req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader("{}"))
			if tc.header != "" || tc.presentEmptyHeader {
					req.Header.Set("Authorization", tc.header)
				}
			req.Header.Set("Content-Type", "application/json")
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "session", Value: tc.cookie})
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if impl.calls-before != boolInt(tc.want) {
				t.Errorf("service calls increased by %d, want %t", impl.calls-before, tc.want)
			}
			if tc.want {
				if rec.Code != http.StatusNoContent {
					t.Errorf("accepted status %d, want %d", rec.Code, http.StatusNoContent)
				}
			} else {
				status := http.StatusUnauthorized
				if tc.problemCode == "decode_payload" {
					status = http.StatusBadRequest
				}
				assertProblem(t, rec, status, tc.problemCode, tc.problemDetail)
			}
			if impl.jwt-beforeJWT != tc.wantJWT || impl.key-beforeKey != tc.wantKey {
				t.Errorf("callback delta JWT=%d APIKey=%d, want %d and %d", impl.jwt-beforeJWT, impl.key-beforeKey, tc.wantJWT, tc.wantKey)
			}
		})
	}
	if impl.jwt != 5 || impl.key != 4 {
		t.Errorf("auth callbacks JWT=%d APIKey=%d, want 5 and 4", impl.jwt, impl.key)
	}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func assertProblem(t *testing.T, rec *httptest.ResponseRecorder, status int, code, detail string) {
	t.Helper()
	if rec.Code != status {
		t.Errorf("rejected status %d, want %d: %s", rec.Code, status, rec.Body.String())
	}
	var problem loomhttp.ProblemResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Errorf("decode problem response %q: %v", rec.Body.String(), err)
		return
	}
	if problem.Code != code {
		t.Errorf("problem code %q, want %q", problem.Code, code)
	}
	if problem.Detail != detail {
		t.Errorf("problem detail %q, want %q", problem.Detail, detail)
	}
}
`

const httpSecurityScopeMatrixHarness = `package scope_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apisvc "example.com/scope/gen/api_scope"
	apiserver "example.com/scope/gen/http/api_scope/server"
	server "example.com/scope/gen/http/scope/server"
	svc "example.com/scope/gen/scope"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

type implementation struct {
	basic, jwt, key int
	calls           map[string]int
}

func (s *implementation) BasicAuth(ctx context.Context, user, pass string, _ *security.BasicScheme) (context.Context, error) {
	s.basic++
	if user == "user" && pass == "pass" {
		return ctx, nil
	}
	return ctx, loom.PermanentError("unauthorized", "invalid credentials")
}
func (s *implementation) JWTAuth(ctx context.Context, token string, _ *security.JWTScheme) (context.Context, error) {
	s.jwt++
	if token == "jwt-ok" {
		return ctx, nil
	}
	return ctx, loom.PermanentError("unauthorized", "invalid credentials")
}
func (s *implementation) APIKeyAuth(ctx context.Context, token string, _ *security.APIKeyScheme) (context.Context, error) {
	s.key++
	if token == "key-ok" {
		return ctx, nil
	}
	return ctx, loom.PermanentError("unauthorized", "invalid credentials")
}
func (s *implementation) Read(context.Context, *apisvc.ReadPayload) error {
	s.calls["read"]++
	return nil
}
func (s *implementation) Inherited(context.Context, *svc.InheritedPayload) error {
	s.calls["inherited"]++
	return nil
}
func (s *implementation) Override(context.Context, *svc.OverridePayload) error {
	s.calls["override"]++
	return nil
}
func (s *implementation) Alternatives(context.Context, *svc.AlternativesPayload) error {
	s.calls["alternatives"]++
	return nil
}
func (s *implementation) Open(context.Context, *svc.OpenPayload) error {
	s.calls["open"]++
	return nil
}

func TestHTTPSecurityScopeMatrix(t *testing.T) {
	impl := &implementation{calls: make(map[string]int)}
	mux := loomhttp.NewMuxer()
	apiserver.Mount(mux, apiserver.New(apisvc.NewEndpoints(impl), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	server.Mount(mux, server.New(svc.NewEndpoints(impl), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	for _, tc := range []struct {
		name, path, auth string
		want             bool
		basic, jwt, key  int
	}{
		{name: "api inherited basic", path: "/read", auth: "Basic dXNlcjpwYXNz", want: true, basic: 1},
		{name: "api rejected basic", path: "/read", auth: "Basic dXNlcjpiYWQ=", basic: 1},
		{name: "service inherited jwt", path: "/inherited", auth: "Bearer jwt-ok", want: true, jwt: 1},
		{name: "service inherited rejected", path: "/inherited", auth: "Bearer wrong", jwt: 1},
		{name: "method override api key", path: "/override", auth: "key-ok", want: true, key: 1},
		{name: "method override rejected", path: "/override", auth: "bad-key", key: 1},
		{name: "alternative basic", path: "/alternatives", auth: "Basic dXNlcjpwYXNz", want: true, basic: 1},
		{name: "alternative jwt", path: "/alternatives", auth: "Bearer jwt-ok", want: true, basic: 1, jwt: 1},
		{name: "alternative rejected", path: "/alternatives", auth: "Bearer wrong", basic: 1, jwt: 1},
		{name: "no security", path: "/open", want: true},
		{name: "no security ignores invalid credentials", path: "/open", auth: "Bearer wrong", want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := totalCalls(impl.calls)
			beforeBasic, beforeJWT, beforeKey := impl.basic, impl.jwt, impl.key
			req := httptest.NewRequest(http.MethodGet, tc.path, strings.NewReader("{}"))
			if tc.auth != "" {
				req.Header.Set("Authorization", tc.auth)
			}
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			delta := totalCalls(impl.calls) - before
			if delta != boolInt(tc.want) {
				t.Errorf("service call delta %d, want %t", delta, tc.want)
			}
			if tc.want {
				if rec.Code != http.StatusNoContent {
					t.Errorf("accepted status %d, want %d", rec.Code, http.StatusNoContent)
				}
			} else {
				assertUnauthorizedProblem(t, rec)
			}
			if impl.basic-beforeBasic != tc.basic || impl.jwt-beforeJWT != tc.jwt || impl.key-beforeKey != tc.key {
				t.Errorf("callback delta Basic=%d JWT=%d APIKey=%d, want %d, %d, %d", impl.basic-beforeBasic, impl.jwt-beforeJWT, impl.key-beforeKey, tc.basic, tc.jwt, tc.key)
			}
		})
	}
	if impl.basic != 5 || impl.jwt != 4 || impl.key != 2 {
		t.Errorf("auth callbacks Basic=%d JWT=%d APIKey=%d, want 5, 4, 2", impl.basic, impl.jwt, impl.key)
	}
	if impl.calls["read"] != 1 || impl.calls["inherited"] != 1 || impl.calls["override"] != 1 || impl.calls["alternatives"] != 2 || impl.calls["open"] != 2 {
		t.Errorf("service calls %#v, want one each except two alternatives and two open calls", impl.calls)
	}
}

func totalCalls(calls map[string]int) int {
	total := 0
	for _, count := range calls {
		total += count
	}
	return total
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func assertUnauthorizedProblem(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("rejected status %d, want %d: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	var problem loomhttp.ProblemResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Errorf("decode problem response %q: %v", rec.Body.String(), err)
		return
	}
	if problem.Code != "unauthorized" {
		t.Errorf("problem code %q, want unauthorized", problem.Code)
	}
	if problem.Detail != "invalid credentials" {
		t.Errorf("problem detail %q, want %q", problem.Detail, "invalid credentials")
	}
}
`
