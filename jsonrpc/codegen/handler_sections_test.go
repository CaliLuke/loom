package codegen

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
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
	require.Contains(t, code, `s.handleBatch(w, r)`)
	require.Contains(t, code, `s.handleSingle(w, r)`)
	require.Equal(t, 2, strings.Count(code, `jsonrpcEnvelopeDecodeError(err)`))
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-server-handler-single-method.golden"), code)
}

func TestJSONRPCEnvelopeDecodeErrorClassification(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcSingleMethodDSL)

	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-encode-error")
	require.Contains(t, code, `func jsonrpcEnvelopeDecodeError(err error) (jsonrpc.Code, string, any)`)
	require.Contains(t, code, `errors.As(err, &serviceError) && serviceError.Name == loom.RequestBodyTooLarge`)
	require.Contains(t, code, `return jsonrpcErrorCodeForServiceError(serviceError), loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err)`)
	require.Contains(t, code, `return jsonrpc.ParseError, "Parse error", nil`)
}

func TestJSONRPCProcessRequestBodyValidatesAndDispatches(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcSingleMethodDSL)

	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-handler")
	require.Less(t, findIndex(t, code, `if req.Invalid {`), findIndex(t, code, `if req.JSONRPC != "2.0" {`))
	require.Contains(t, code, `case "add":`)
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

// TestJSONRPCObserverReasons asserts that the JSON-RPC generator emits each
// stable reason the plan requires somewhere in the generated server code.
// This is a source-text contract: a future refactor that drops an emission
// site silently would break here even if the runtime test suite still
// passes. ReasonPanic is covered by the runtime panic test on
// loomtransport.RequestObserver — it is not emitted as a literal in the
// generator because obs.End()'s deferred recover() handles it.
func TestJSONRPCObserverReasons(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcSingleMethodDSL)
	files := ServerFiles("", CreateJSONRPCServices(root))
	handler := fileSectionCode(t, files, "server.go", "jsonrpc-server-handler")
	perEndpoint := fileSectionCode(t, files, "server.go", "jsonrpc-server-handler-init")
	combined := handler + "\n" + perEndpoint

	for _, reason := range []string{
		"ReasonInvalidJSONRPCEnvelope",
		"ReasonInvalidJSONRPCBatch",
		"ReasonInvalidJSONRPCMethod",
		"ReasonUnsupportedMethod",
		"ReasonInvalidJSONRPCParams",
		"ReasonHandlerError",
		"ReasonResponseWriteFailed",
	} {
		require.Containsf(t, combined, "loomtransport."+reason, "generated JSON-RPC handler must emit %s at least once", reason)
	}
	require.Contains(t, handler, "loomtransport.BeginJSONRPCRequest(", "generated handler must Begin a JSON-RPC observer")
	require.Contains(t, handler, "defer obs.End()", "generated handler must defer obs.End() so ReasonPanic is emitted on recovered panics")
	require.Contains(t, handler, "SetJSONRPC(", "generated handler must enrich the observer with JSON-RPC envelope fields after decode")
}

func TestJSONRPCBatchWriterHelperSection(t *testing.T) {
	section := codegen.NewJenniferSection("test-batch-writer", func(stmt *jen.Statement) {
		addJSONRPCBatchWriterSection(stmt)
	})
	code := codegen.SectionCode(t, section)

	require.Contains(t, code, `type batchWriter struct`)
	require.NotContains(t, code, "statusCode int")
	require.Contains(t, code, "JSON-RPC batch items do not control the outer HTTP status")
	require.Contains(t, code, "write JSON-RPC batch delimiter: %w")
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-batch-writer.golden"), code)
}

func TestJSONRPCBatchHandlingDecodesElementsIndependently(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcSingleMethodDSL)

	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-handler")

	require.Contains(t, code, `for _, rawReq := range rawReqs {`)
	require.Contains(t, code, `json.Unmarshal(rawReq, &req)`)
	require.NotContains(t, code, `var reqs []jsonrpc.RawRequest`)
	require.Contains(t, code, `failed to close JSON-RPC batch response: %w`)
	require.Contains(t, code, `loomtransport.ReasonResponseWriteFailed`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-server-handler-single-method.golden"), code)
}

func TestJSONRPCInvalidRequestWithoutIDStillRespondsWithNullID(t *testing.T) {
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

	require.Contains(t, code, `if !req.HasID {`)
	require.Contains(t, code, `id = nil`)
	require.NotContains(t, code, `if req.ID != nil {`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-server-encode-error.golden"), code)
}
