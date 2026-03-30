package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestServerGRPCInterface(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"unary-rpcs", testdata.UnaryRPCsDSL},
		{"unary-rpc-no-payload", testdata.UnaryRPCNoPayloadDSL},
		{"unary-rpc-no-result", testdata.UnaryRPCNoResultDSL},
		{"unary-rpc-with-errors", testdata.UnaryRPCWithErrorsDSL},
		{"unary-rpc-with-overriding-errors", testdata.UnaryRPCWithOverridingErrorsDSL},
		{"server-streaming-rpc", testdata.ServerStreamingRPCDSL},
		{"client-streaming-rpc", testdata.ClientStreamingRPCDSL},
		{"client-streaming-rpc-with-payload", testdata.ClientStreamingRPCWithPayloadDSL},
		{"bidirectional-streaming-rpc", testdata.BidirectionalStreamingRPCDSL},
		{"bidirectional-streaming-rpc-with-payload", testdata.BidirectionalStreamingRPCWithPayloadDSL},
		{"bidirectional-streaming-rpc-with-errors", testdata.BidirectionalStreamingRPCWithErrorsDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunGRPCDSL(t, c.DSL)
			services := CreateGRPCServices(root)
			fs := ServerFiles("", services)
			require.Len(t, fs, 2)
			sections := fs[0].Section("server-grpc-interface")
			require.NotEmpty(t, sections)
			code := codegen.SectionsCode(t, sections)
			testutil.AssertGo(t, "testdata/golden/server_grpc_interface_"+c.Name+".go.golden", code)
		})
	}
}

func TestServerHandlerInit(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"unary-rpcs", testdata.UnaryRPCsDSL},
		{"unary-rpc-no-payload", testdata.UnaryRPCNoPayloadDSL},
		{"unary-rpc-no-result", testdata.UnaryRPCNoResultDSL},
		{"server-streaming-rpc", testdata.ServerStreamingRPCDSL},
		{"client-streaming-rpc", testdata.ClientStreamingRPCDSL},
		{"client-streaming-rpc-with-payload", testdata.ClientStreamingRPCWithPayloadDSL},
		{"bidirectional-streaming-rpc", testdata.BidirectionalStreamingRPCDSL},
		{"bidirectional-streaming-rpc-with-payload", testdata.BidirectionalStreamingRPCWithPayloadDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunGRPCDSL(t, c.DSL)
			services := CreateGRPCServices(root)
			fs := ServerFiles("", services)
			require.Len(t, fs, 2)
			sections := fs[0].Section("grpc-handler-init")
			require.NotEmpty(t, sections)
			code := codegen.SectionsCode(t, sections)
			testutil.AssertGo(t, "testdata/golden/server_handler_init_"+c.Name+".go.golden", code)
		})
	}
}

func TestRequestDecoder(t *testing.T) {
	assertGRPCSectionGolden(t, grpcCodecCases("request-decoder-", grpcCodecDSLs), func(services *ServicesData) []*codegen.File {
		return ServerFiles("", services)
	}, "request-decoder", "request_decoder_")
}

func TestResponseEncoder(t *testing.T) {
	assertGRPCSectionGolden(t, []grpcDSLTestCase{
		{Name: "response-encoder-empty-result", DSL: testdata.UnaryRPCNoResultDSL},
		{Name: "response-encoder-result-with-views", DSL: testdata.MessageResultTypeWithViewsDSL},
		{Name: "response-encoder-result-with-explicit-view", DSL: testdata.MessageResultTypeWithExplicitViewDSL},
		{Name: "response-encoder-result-array", DSL: testdata.MessageArrayDSL},
		{Name: "response-encoder-result-primitive", DSL: testdata.UnaryRPCNoPayloadDSL},
		{Name: "response-encoder-result-with-metadata", DSL: testdata.MessageWithMetadataDSL},
		{Name: "response-encoder-result-with-validate", DSL: testdata.MessageWithValidateDSL},
		{Name: "response-encoder-result-collection", DSL: testdata.MessageResultTypeCollectionDSL},
	}, func(services *ServicesData) []*codegen.File {
		return ServerFiles("", services)
	}, "response-encoder", "response_encoder_")
}
