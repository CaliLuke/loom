package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
)

func TestJSONRPCServerCORSUsesDesignPolicy(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcCORSPolicyDSL)
	files := ServerFiles("", CreateJSONRPCServices(root))

	initCode := fileSectionCode(t, files, "server.go", "jsonrpc-server-init")
	require.Contains(t, initCode, `loomhttp.CORSHandler(loomhttp.CORSPolicy{`)
	require.NotContains(t, initCode, "NewCrossOriginProtection")
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-server-init-cors.golden"), initCode)

	mountCode := fileSectionCode(t, files, "server.go", "jsonrpc-server-mount")
	require.Contains(t, mountCode, "loomhttp.HandleCORSPreflight")
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-server-mount-cors.golden"), mountCode)
}

func TestJSONRPCServerWithoutCORSKeepsSecureDefault(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcDefaultOriginProtectionDSL)
	files := ServerFiles("", CreateJSONRPCServices(root))

	initCode := fileSectionCode(t, files, "server.go", "jsonrpc-server-init")
	require.Contains(t, initCode, "http.NewCrossOriginProtection()")
	require.Contains(t, initCode, "http.HandlerFunc(s.serveHTTP)")
	require.NotContains(t, initCode, "loomhttp.CORSHandler")

	mountCode := fileSectionCode(t, files, "server.go", "jsonrpc-server-mount")
	require.Contains(t, mountCode, `mux.Handle("POST", "/rpc", h.ServeHTTP)`)
	require.NotContains(t, mountCode, `mux.Handle("OPTIONS"`)
}

func TestJSONRPCServerRuntimeCORSUsesConstructorPolicy(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcRuntimeCORSPolicyDSL)
	files := ServerFiles("", CreateJSONRPCServices(root))

	initCode := fileSectionCode(t, files, "server.go", "jsonrpc-server-init")
	require.Contains(t, initCode, "corsPolicy loomhttp.RuntimeCORSPolicy")
	require.Contains(t, initCode, "corsPolicy.Handler(http.HandlerFunc(s.serveHTTP))")
	require.NotContains(t, initCode, "NewCrossOriginProtection")

	mountCode := fileSectionCode(t, files, "server.go", "jsonrpc-server-mount")
	require.Contains(t, mountCode, `mux.Handle("POST", "/rpc", h.ServeHTTP)`)
	require.NotContains(t, mountCode, "h.Handler.ServeHTTP")
	require.Contains(t, mountCode, "h.corsPolicy.HandlePreflight")
}

func TestJSONRPCRuntimeCORSGeneratedModuleCompiles(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcRuntimeCORSPolicyDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcruntimecors", root)
	serverDirs, err := filepath.Glob(filepath.Join(dir, "gen", "jsonrpc", "*", "server"))
	require.NoError(t, err)
	require.Len(t, serverDirs, 1)
	require.NoError(t, os.WriteFile(
		filepath.Join(serverDirs[0], "runtime_cors_mount_test.go"),
		[]byte(jsonrpcRuntimeCORSMountTest),
		0o644,
	))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}

func TestJSONRPCRuntimeCORSAppliesToStreamingTransports(t *testing.T) {
	tests := []struct {
		name string
		dsl  func()
	}{
		{name: "sse only", dsl: testdata.JSONRPCSSEEventsStreamDSL},
		{name: "mixed", dsl: jsonrpcMixedInitializeAndEventsStreamDSL},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := RunJSONRPCDSL(t, test.dsl)
			root.API.JSONRPC.Services[0].CORS = &expr.HTTPCORSExpr{Runtime: true}
			files := ServerFiles("", CreateJSONRPCServices(root))
			initCode := fileSectionCode(t, files, "server.go", "jsonrpc-server-init")
			require.Contains(t, initCode, "corsPolicy loomhttp.RuntimeCORSPolicy")
			require.Contains(t, initCode, "corsPolicy.Handler")
			require.NotContains(t, initCode, "NewCrossOriginProtection")
			mountCode := fileSectionCode(t, files, "server.go", "jsonrpc-server-mount")
			require.Contains(t, mountCode, "h.corsPolicy.HandlePreflight")
		})
	}
}

func TestJSONRPCWildcardCORSProvidesExplicitOptOut(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcWildcardCORSPolicyDSL)
	initCode := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-init")

	require.Contains(t, initCode, `Pattern: "*"`)
	require.NotContains(t, initCode, "NewCrossOriginProtection")
}

func TestJSONRPCServerInheritsAPICORSPolicy(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcAPICORSPolicyDSL)
	initCode := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-init")

	require.Contains(t, initCode, `"https://api.example.com"`)
	require.NotContains(t, initCode, "NewCrossOriginProtection")
}

func TestJSONRPCCORSAppliesToEveryServerTransport(t *testing.T) {
	cases := []struct {
		name    string
		dsl     func()
		handler string
	}{
		{name: "plain", dsl: jsonrpcDefaultOriginProtectionDSL, handler: "http.HandlerFunc(s.serveHTTP)"},
		{name: "sse", dsl: testdata.JSONRPCSSEEventsStreamDSL, handler: "http.HandlerFunc(s.handleSSE)"},
		{name: "mixed", dsl: jsonrpcMixedInitializeAndEventsStreamDSL, handler: "http.HandlerFunc(s.serveHTTP)"},
		{name: "websocket", dsl: jsonrpcWebSocketRuntimeDSL, handler: "http.HandlerFunc(s.serveHTTP)"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			root := RunJSONRPCDSL(t, test.dsl)
			root.API.JSONRPC.Services[0].CORS = &expr.HTTPCORSExpr{Origins: []*expr.HTTPCORSOriginExpr{{Pattern: "*"}}}
			initCode := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-init")

			require.Contains(t, initCode, "loomhttp.CORSHandler")
			require.Contains(t, initCode, test.handler)
			require.NotContains(t, initCode, "NewCrossOriginProtection")
		})
	}
}

var jsonrpcCORSPolicyDSL = func() {
	dsl.Service("JSONRPCCORS", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
			dsl.CORS(func() {
				dsl.Origin("https://app.example.com", func() {
					dsl.Methods("POST")
					dsl.Headers("Content-Type", "Authorization")
					dsl.Credentials()
				})
			})
		})
		dsl.Method("Call", func() {
			dsl.Payload(dsl.String)
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
}

var jsonrpcDefaultOriginProtectionDSL = func() {
	dsl.Service("JSONRPCDefaultOriginProtection", func() {
		dsl.JSONRPC(func() { dsl.POST("/rpc") })
		dsl.Method("Call", func() {
			dsl.Payload(dsl.String)
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
}

var jsonrpcRuntimeCORSPolicyDSL = func() {
	dsl.Service("JSONRPCRuntimeCORS", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
			dsl.RuntimeCORS()
		})
		dsl.Method("Call", func() {
			dsl.Payload(dsl.String)
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
}

var jsonrpcWildcardCORSPolicyDSL = func() {
	dsl.Service("JSONRPCWildcardCORS", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
			dsl.CORS(func() { dsl.Origin("*") })
		})
		dsl.Method("Call", func() {
			dsl.Payload(dsl.String)
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
}

var jsonrpcAPICORSPolicyDSL = func() {
	dsl.API("JSONRPC CORS", func() {
		dsl.JSONRPC(func() {
			dsl.CORS(func() { dsl.Origin("https://api.example.com") })
		})
	})
	dsl.Service("JSONRPCAPICORS", func() {
		dsl.JSONRPC(func() { dsl.POST("/rpc") })
		dsl.Method("Call", func() {
			dsl.Payload(dsl.String)
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
}

const jsonrpcRuntimeCORSMountTest = `package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/jsonrpc"
	"github.com/stretchr/testify/require"
)

func TestMountedServerUsesEffectiveHandler(t *testing.T) {
	policy, err := loomhttp.NewRuntimeCORSPolicy(loomhttp.CORSPolicy{Origins: []loomhttp.CORSOrigin{{
		Pattern:     "https://app.example.com",
		Expose:      []string{"X-Request-Id"},
		Credentials: true,
	}}})
	require.NoError(t, err)

	var applicationCalls atomic.Int32
	server := &Server{
		Call: func(_ context.Context, _ *http.Request, _ *jsonrpc.RawRequest, w http.ResponseWriter) error {
			applicationCalls.Add(1)
			w.WriteHeader(http.StatusAccepted)
			return nil
		},
		Methods:    []string{"Call"},
		decoder:    loomhttp.RequestDecoder,
		encoder:    loomhttp.ResponseEncoder,
		errhandler: func(context.Context, http.ResponseWriter, error) {},
		corsPolicy: policy,
	}
	server.Handler = policy.Handler(http.HandlerFunc(server.serveHTTP))

	var middlewareCalls atomic.Int32
	server.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalls.Add(1)
			next.ServeHTTP(w, r)
		})
	})

	mux := loomhttp.NewMuxer()
	Mount(mux, server)
	// Transport extensions such as MCP add these route-local methods around
	// the generated server's public, effective ServeHTTP entry point.
	mux.Handle(http.MethodGet, "/rpc", server.ServeHTTP)
	mux.Handle(http.MethodDelete, "/rpc", server.ServeHTTP)

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		t.Run("allowed "+method, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(method, "/rpc", strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"Call\",\"params\":\"ok\"}"))
			request.Header.Set("Origin", "https://app.example.com")
			mux.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusAccepted, recorder.Code)
			require.Equal(t, "https://app.example.com", recorder.Header().Get("Access-Control-Allow-Origin"))
			require.Equal(t, "true", recorder.Header().Get("Access-Control-Allow-Credentials"))
			require.Equal(t, "X-Request-Id", recorder.Header().Get("Access-Control-Expose-Headers"))
		})
	}
	require.EqualValues(t, 3, applicationCalls.Load())
	require.EqualValues(t, 3, middlewareCalls.Load())

	t.Run("disallowed origin passes through", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"Call\",\"params\":\"ok\"}"))
		request.Header.Set("Origin", "https://evil.example.com")
		mux.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusAccepted, recorder.Code)
		require.Empty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("preflight is route local and terminal", func(t *testing.T) {
		applicationCallsBefore := applicationCalls.Load()
		middlewareCallsBefore := middlewareCalls.Load()
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodOptions, "/rpc", nil)
		request.Header.Set("Origin", "https://app.example.com")
		request.Header.Set("Access-Control-Request-Method", http.MethodPost)
		mux.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusNoContent, recorder.Code)
		require.Equal(t, http.MethodPost, recorder.Header().Get("Access-Control-Allow-Methods"))
		require.Equal(t, applicationCallsBefore, applicationCalls.Load())
		require.Equal(t, middlewareCallsBefore, middlewareCalls.Load())
	})
}
`
