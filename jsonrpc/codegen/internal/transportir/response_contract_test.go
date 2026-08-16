package transportir

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestAnalyzeResponseContractCases(t *testing.T) {
	root := expr.RunDSL(t, responseContractDSL)
	endpoint := root.API.JSONRPC.Services[0].HTTPEndpoints[0]

	analysis := AnalyzeResponseContractCases(endpoint)
	require.True(t, analysis.Supported())
	require.Equal(t, []*ResponseContractCase{
		{
			ID:         "tools.call.success",
			Kind:       ResponseContractSuccess,
			ResultType: "string",
			HasResult:  true,
		},
		{
			ID:            "tools.call.error.forbidden.4403",
			Kind:          ResponseContractError,
			ErrorCode:     4403,
			ErrorName:     "forbidden",
			ErrorDataType: "ToolError",
		},
		{
			ID:            "tools.call.error.internal.-32603",
			Kind:          ResponseContractError,
			ErrorCode:     -32603,
			ErrorName:     "internal",
			ErrorDataType: "jsonrpc.ErrorData",
		},
		{
			ID:   "tools.call.notification",
			Kind: ResponseContractNotification,
		},
	}, analysis.Cases)
}

func TestAnalyzeResponseContractCasesSupportsServerSSE(t *testing.T) {
	root := expr.RunDSL(t, responseContractSSEDSL)
	endpoint := root.API.JSONRPC.Services[0].HTTPEndpoints[0]

	analysis := AnalyzeResponseContractCases(endpoint)
	require.True(t, analysis.Supported())
	require.Equal(t, &ResponseContractStream{
		Transport: "sse",
		Terminal:  "final_response",
	}, analysis.Cases[0].Stream)
	require.Equal(t, &ResponseContractStream{
		Transport: "sse",
		Terminal:  "suppressed",
	}, analysis.Cases[len(analysis.Cases)-1].Stream)
}

func TestAnalyzeResponseContractCasesRejectsWebSocketStream(t *testing.T) {
	root := expr.RunDSL(t, responseContractWebSocketDSL)
	endpoint := root.API.JSONRPC.Services[0].HTTPEndpoints[0]

	analysis := AnalyzeResponseContractCases(endpoint)
	require.False(t, analysis.Supported())
	require.Equal(t, []ResponseContractLimitation{{
		Code:   ResponseContractStreaming,
		Detail: "JSON-RPC response contracts currently support unary and server-SSE endpoints",
	}}, analysis.Limitations)
}

func responseContractDSL() {
	toolError := Type("ToolError", func() {
		ErrorName("name", String)
		Attribute("message", String)
		Required("name", "message")
	})
	Service("tools", func() {
		JSONRPC(func() { POST("/rpc") })
		Method("call", func() {
			Result(String)
			Error("forbidden", toolError)
			Error("internal")
			JSONRPC(func() {
				Response("forbidden", 4403)
				Response("internal", RPCInternalError)
			})
		})
	})
}

func responseContractSSEDSL() {
	Service("events", func() {
		JSONRPC(func() { POST("/rpc") })
		Method("watch", func() {
			StreamingResult(String)
			JSONRPC(func() { ServerSentEvents() })
		})
	})
}

func responseContractWebSocketDSL() {
	Service("events", func() {
		JSONRPC(func() { GET("/rpc") })
		Method("watch", func() {
			StreamingResult(String)
			JSONRPC(func() {})
		})
	})
}
