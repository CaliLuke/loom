package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestClientEndpointInit(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"unary-rpcs", testdata.UnaryRPCsDSL},
		{"unary-rpc-no-payload", testdata.UnaryRPCNoPayloadDSL},
		{"unary-rpc-no-result", testdata.UnaryRPCNoResultDSL},
		{"unary-rpc-with-errors", testdata.UnaryRPCWithErrorsDSL},
		{"unary-rpc-acronym", testdata.UnaryRPCAcronymDSL},
		{"server-streaming-rpc", testdata.ServerStreamingRPCDSL},
		{"client-streaming-rpc", testdata.ClientStreamingRPCDSL},
		{"client-streaming-rpc-no-result", testdata.ClientStreamingNoResultDSL},
		{"client-streaming-rpc-with-payload", testdata.ClientStreamingRPCWithPayloadDSL},
		{"bidirectional-streaming-rpc", testdata.BidirectionalStreamingRPCDSL},
		{"bidirectional-streaming-rpc-with-payload", testdata.BidirectionalStreamingRPCWithPayloadDSL},
		{"bidirectional-streaming-rpc-with-errors", testdata.BidirectionalStreamingRPCWithErrorsDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunGRPCDSL(t, c.DSL)
			services := CreateGRPCServices(root)
			fs := ClientFiles("", services)
			require.Len(t, fs, 2)
			sections := fs[0].Section("client-endpoint-init")
			if len(sections) == 0 {
				t.Fatalf("got zero sections, expected at least one")
			}
			code := codegen.SectionsCode(t, sections)
			testutil.AssertGo(t, "testdata/golden/client_endpoint_init_"+c.Name+".go.golden", code)
		})
	}
}

func TestRequestEncoder(t *testing.T) {
	assertGRPCSectionGolden(t, grpcCodecCases("request-encoder-", grpcCodecDSLs), func(services *ServicesData) []*codegen.File {
		return ClientFiles("", services)
	}, "request-encoder", "request_encoder_")
}

func TestResponseDecoder(t *testing.T) {
	assertGRPCSectionGolden(t, grpcCodecCases("response-decoder-", grpcResultCodecDSLs), func(services *ServicesData) []*codegen.File {
		return ClientFiles("", services)
	}, "response-decoder", "response_decoder_")
}
