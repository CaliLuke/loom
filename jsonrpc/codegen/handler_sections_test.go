package codegen

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
)

// jsonrpcSingleMethodDSL is the shared plain JSON-RPC design (one service, one
// unary method) used by the handler section tests below. The rendered handler
// sections do not depend on the API name, so the tests share one fixture.
var jsonrpcSingleMethodDSL = func() {
	dsl.API("jsonrpc-handler-sections-test", func() {
		dsl.JSONRPC(func() {})
	})
	dsl.Service("calc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("add", func() {
			dsl.Payload(func() {
				dsl.ID("id", dsl.String)
			})
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
}

func TestJSONRPCHandlerSectionRoutesBufferedRequests(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcSingleMethodDSL)

	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-handler")
	require.Contains(t, code, `jsonrpc.ServeHTTP(`)
	require.Contains(t, code, `jsonrpc.HTTPHandlerSpec{`)
	require.Contains(t, code, `Dispatch:      s.dispatchHTTP`)
	require.NotContains(t, code, `handleBatch`)
	require.NotContains(t, code, `handleSingle`)
	require.NotContains(t, code, `batchWriter`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-server-handler-single-method.golden"), code)
}

func TestJSONRPCEnvelopeDecodeErrorClassificationIsRuntimeOwned(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcSingleMethodDSL)

	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-encode-error")
	require.NotContains(t, code, `jsonrpcEnvelopeDecodeError`)
}

func TestJSONRPCProcessRequestBodyValidatesAndDispatches(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcSingleMethodDSL)

	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-handler")
	require.Contains(t, code, `func (s *Server) dispatchHTTP(`)
	require.Contains(t, code, `case "add":`)
	require.NotContains(t, code, `if req.Invalid`)
	require.NotContains(t, code, `if req.JSONRPC`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-server-handler-single-method.golden"), code)
}

func findIndex(t *testing.T, value, substr string) int {
	t.Helper()
	index := strings.Index(value, substr)
	require.NotEqualf(t, -1, index, "expected %q in generated code", substr)
	return index
}

func TestJSONRPCHandlerInitDecodesParamsWithGeneratedDecoderSignature(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcSingleMethodDSL)

	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-handler-init")

	require.Contains(t, code, `decodeParams := DecodeAddRequest(mux, decoder)`)
	require.Contains(t, code, `params, err := decodeParams(r, req)`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-server-handler-init-single-method.golden"), code)
}

func TestJSONRPCSSEHandlerInitHoistsRequestDecoder(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-handler-init")

	decodeDecl := findIndex(t, code, `decodeParams := DecodeStreamRequest(mux, decoder)`)
	closure := findIndex(t, code, `return func(ctx context.Context`)
	decodeUse := findIndex(t, code, `params, err := decodeParams(r, req)`)

	require.Less(t, decodeDecl, closure)
	require.Less(t, closure, decodeUse)
}

func TestJSONRPCHandlerDelegatesProtocolLifecycle(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcSingleMethodDSL)
	files := ServerFiles("", CreateJSONRPCServices(root))
	handler := fileSectionCode(t, files, "server.go", "jsonrpc-server-handler")
	perEndpoint := fileSectionCode(t, files, "server.go", "jsonrpc-server-handler-init")

	require.Contains(t, handler, "jsonrpc.ServeHTTP(")
	for _, runtimeOwned := range []string{
		"loomtransport.BeginJSONRPCRequest(",
		"loomtransport.ReasonInvalidJSONRPCEnvelope",
		"loomtransport.ReasonInvalidJSONRPCBatch",
		"loomtransport.ReasonInvalidJSONRPCMethod",
		"loomtransport.ReasonUnsupportedMethod",
		"SetJSONRPC(",
	} {
		require.NotContains(t, handler, runtimeOwned)
	}
	for _, adapterOwned := range []string{
		"ReasonInvalidJSONRPCParams",
		"ReasonHandlerError",
		"ReasonResponseWriteFailed",
	} {
		require.Contains(t, perEndpoint, "loomtransport."+adapterOwned)
	}
}

func TestJSONRPCBatchHandlingDecodesElementsIndependently(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcSingleMethodDSL)

	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-handler")

	require.Contains(t, code, `jsonrpc.ServeHTTP(`)
	require.NotContains(t, code, `for _, rawReq`)
	require.NotContains(t, code, `json.Unmarshal`)
	require.NotContains(t, code, `batchWriter`)
	require.NotContains(t, code, `failed to close JSON-RPC batch response`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-server-handler-single-method.golden"), code)
}

func TestJSONRPCErrorAdapterLeavesNotificationSuppressionToRuntime(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		dsl.API("jsonrpc-invalid-request-test", func() {
			dsl.JSONRPC(func() {})
		})
		dsl.Service("calc", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			dsl.Method("add", func() {
				dsl.Result(dsl.String)
				dsl.JSONRPC(func() {})
			})
		})
	})

	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-encode-error")

	require.Contains(t, code, `jsonrpc.MakeErrorResponse(req.ID, code, message, data)`)
	require.NotContains(t, code, `req.HasID`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-server-encode-error.golden"), code)
}
