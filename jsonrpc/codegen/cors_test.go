package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
)

func TestJSONRPCServerCORSUsesDesignPolicy(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcCORSPolicyDSL)
	files := ServerFiles("", CreateJSONRPCServices(root))

	initCode := fileSectionCode(t, files, "server.go", "jsonrpc-server-init")
	require.Contains(t, initCode, `loomhttp.CORSHandler(loomhttp.CORSPolicy{`)
	require.Contains(t, initCode, `"https://app.example.com"`)
	require.Contains(t, initCode, `Credentials: true`)
	require.NotContains(t, initCode, "NewCrossOriginProtection")

	mountCode := fileSectionCode(t, files, "server.go", "jsonrpc-server-mount")
	require.Contains(t, mountCode, `mux.Handle("OPTIONS", "/rpc"`)
	require.Contains(t, mountCode, "loomhttp.HandleCORSPreflight")
	require.Contains(t, mountCode, `[]string{"POST"}`)
}

func TestJSONRPCServerWithoutCORSKeepsSecureDefault(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcDefaultOriginProtectionDSL)
	files := ServerFiles("", CreateJSONRPCServices(root))

	initCode := fileSectionCode(t, files, "server.go", "jsonrpc-server-init")
	require.Contains(t, initCode, "http.NewCrossOriginProtection()")
	require.NotContains(t, initCode, "loomhttp.CORSHandler")

	mountCode := fileSectionCode(t, files, "server.go", "jsonrpc-server-mount")
	require.NotContains(t, mountCode, `mux.Handle("OPTIONS"`)
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
		{name: "plain", dsl: jsonrpcDefaultOriginProtectionDSL, handler: "http.HandlerFunc(s.ServeHTTP)"},
		{name: "sse", dsl: testdata.JSONRPCSSEEventsStreamDSL, handler: "http.HandlerFunc(s.handleSSE)"},
		{name: "mixed", dsl: jsonrpcMixedInitializeAndEventsStreamDSL, handler: "http.HandlerFunc(s.ServeHTTP)"},
		{name: "websocket", dsl: jsonrpcWebSocketRuntimeDSL, handler: "http.HandlerFunc(s.ServeHTTP)"},
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
