package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestServerResponseContractCases(t *testing.T) {
	root := RunGRPCDSL(t, testdata.UnaryRPCWithErrorsDSL)
	services := CreateGRPCServices(root)
	file := ServerFiles("example.com/widgets/gen", services)[0]
	generated := codegen.SectionsCode(t, file.Section("server-response-contract"))

	require.Contains(t, generated, "func MethodUnaryRPCWithErrorsResponseContractCases() []loomgrpc.ResponseContractCase")
	require.Contains(t, generated, "Kind: loomgrpc.ResponseContractSuccess")
	require.Contains(t, generated, "StatusCode: codes.OK")
	require.Contains(t, generated, `MessageType: "service_unary_rpc_with_errors.MethodUnaryRPCWithErrorsResponse"`)
	require.Contains(t, generated, "Kind: loomgrpc.ResponseContractError")
	require.Contains(t, generated, `ErrorName: "internal"`)
	require.Contains(t, generated, `DetailType: "service_unary_rpc_with_errors.MethodUnaryRPCWithErrorsInternalError"`)

	aggregate := codegen.SectionsCode(t, file.Section("server-response-contracts"))
	require.Contains(t, aggregate, "func ResponseContractCases() []loomgrpc.ResponseContractCase")
	require.Contains(t, aggregate, "cases = append(cases, MethodUnaryRPCWithErrorsResponseContractCases()...)")
}

func TestServerResponseContractCasesWarnForUnsupportedStream(t *testing.T) {
	root := RunGRPCDSL(t, testdata.BidirectionalStreamingRPCDSL)
	file := ServerFiles("example.com/widgets/gen", CreateGRPCServices(root))[0]
	require.Contains(t, file.Warnings, "gRPC response contract omitted for ServiceBidirectionalStreamingRPC.MethodBidirectionalStreamingRPC: streaming: gRPC response contracts currently support unary and server-streaming endpoints")
}
