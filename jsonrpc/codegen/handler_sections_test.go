package codegen

import (
	"strings"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

func TestJSONRPCHandlerSectionRoutesBufferedRequests(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		dsl.API("jsonrpc-handler-routing-test", func() {
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
	})

	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-handler")
	require.Contains(t, code, `bufReader := bufio.NewReader(r.Body)`)
	require.Contains(t, code, `peek, err := bufReader.Peek(1)`)
	require.Contains(t, code, `s.handleBatch(w, r)`)
	require.Contains(t, code, `s.handleSingle(w, r)`)
}

func TestJSONRPCProcessRequestBodyValidatesAndDispatches(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		dsl.API("jsonrpc-process-request-test", func() {
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
	})

	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-handler")
	require.Contains(t, code, `if req.Invalid {`)
	require.Contains(t, code, `if req.JSONRPC != "2.0" {`)
	require.Contains(t, code, `if req.Method == "" {`)
	require.Less(t, findIndex(t, code, `if req.Invalid {`), findIndex(t, code, `if req.JSONRPC != "2.0" {`))
	require.Contains(t, code, `switch req.Method {`)
	require.Contains(t, code, `case "add":`)
	require.Contains(t, code, `jsonrpc.MethodNotFound`)
}

func findIndex(t *testing.T, value, substr string) int {
	t.Helper()
	index := strings.Index(value, substr)
	require.NotEqualf(t, -1, index, "expected %q in generated code", substr)
	return index
}

func TestJSONRPCHandlerInitDecodesParamsWithGeneratedDecoderSignature(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		dsl.API("jsonrpc-handler-decode-test", func() {
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
	})

	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-handler-init")

	require.Contains(t, code, `decodeParams := DecodeAddRequest(mux, decoder)`)
	require.Contains(t, code, `params, err := decodeParams(r, req)`)
}

// TestJSONRPCObserverReasons asserts that the JSON-RPC generator emits each
// stable reason the plan requires somewhere in the generated server code.
// This is a source-text contract: a future refactor that drops an emission
// site silently would break here even if the runtime test suite still
// passes. ReasonPanic is covered by the runtime panic test on
// loomtransport.RequestObserver — it is not emitted as a literal in the
// generator because obs.End()'s deferred recover() handles it.
func TestJSONRPCObserverReasons(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		dsl.API("jsonrpc-observer-reasons-test", func() {
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
	})
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
	section := codegen.MustJenniferSection("test-batch-writer", func(stmt *jen.Statement) {
		addJSONRPCBatchWriterSection(stmt)
	})
	code := codegen.SectionCode(t, section)

	require.Contains(t, code, `type batchWriter struct`)
	require.Contains(t, code, `func (rb *batchWriter) Header() http.Header`)
	require.Contains(t, code, `func (rb *batchWriter) WriteHeader(statusCode int)`)
	require.Contains(t, code, `func (rb *batchWriter) Write(data []byte) (int, error)`)
	require.Contains(t, code, `rb.written = true`)
	require.Contains(t, code, `return rb.Writer.Write(data)`)
}

func TestJSONRPCBatchHandlingDecodesElementsIndependently(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		dsl.API("jsonrpc-batch-test", func() {
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
	})

	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-handler")

	require.Contains(t, code, `var rawReqs []json.RawMessage`)
	require.Contains(t, code, `if len(rawReqs) == 0 {`)
	require.Contains(t, code, `jsonrpc.InvalidRequest`)
	require.Contains(t, code, `for _, rawReq := range rawReqs {`)
	require.Contains(t, code, `var req jsonrpc.RawRequest`)
	require.Contains(t, code, `json.Unmarshal(rawReq, &req)`)
	require.NotContains(t, code, `var reqs []jsonrpc.RawRequest`)
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

	require.Contains(t, code, `if req.HasID || code == jsonrpc.InvalidRequest {`)
	require.Contains(t, code, `id := req.ID`)
	require.Contains(t, code, `if !req.HasID {`)
	require.Contains(t, code, `id = nil`)
	require.Contains(t, code, `jsonrpc.MakeErrorResponse(id, code, message, data)`)
	require.NotContains(t, code, `if req.ID != nil {`)
}
