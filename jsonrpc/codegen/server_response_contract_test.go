package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
)

func TestServerResponseContractCases(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonRPCTypedErrorBodyDSL)
	file := ServerFiles("example.com/tools/gen", CreateJSONRPCServices(root))[0]
	generated := codegen.SectionsCode(t, file.Section("server-response-contract"))

	require.Contains(t, generated, "func CallResponseContractCases() []jsonrpc.ResponseContractCase")
	require.Contains(t, generated, `ID: "tools.call.success"`)
	require.Contains(t, generated, "Kind: jsonrpc.ResponseContractSuccess")
	require.Contains(t, generated, `ResultType: "string"`)
	require.Contains(t, generated, `ID: "tools.call.error.forbidden.4403"`)
	require.Contains(t, generated, "Kind: jsonrpc.ResponseContractError")
	require.Contains(t, generated, "ErrorCode: 4403")
	require.Contains(t, generated, `ErrorDataType: "ToolError"`)
	require.Contains(t, generated, `ID: "tools.call.notification"`)
	require.Contains(t, generated, "Kind: jsonrpc.ResponseContractNotification")

	aggregate := codegen.SectionsCode(t, file.Section("server-response-contracts"))
	require.Contains(t, aggregate, "func ResponseContractCases() []jsonrpc.ResponseContractCase")
	require.Contains(t, aggregate, "cases = append(cases, CallResponseContractCases()...)")
}

func TestServerResponseContractCasesWarnForUnsupportedStream(t *testing.T) {
	root := RunJSONRPCDSL(t, responseContractUnsupportedWebSocketDSL)
	file := ServerFiles("example.com/events/gen", CreateJSONRPCServices(root))[0]
	require.Contains(t, file.Warnings, "JSON-RPC response contract omitted for events.watch: streaming: JSON-RPC response contracts currently support unary and server-SSE endpoints")
}
