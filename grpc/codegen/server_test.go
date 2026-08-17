package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/expr"
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
		{"server-streaming-rpc-with-custom-errors", testdata.ServerStreamingWithCustomErrorsDSL},
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
			require.Contains(t, code, "loomgrpc.Serve")
			require.NotContains(t, code, "context.WithValue")
			require.NotContains(t, code, "loomgrpc.NewStatusError")
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

func TestGRPCProjectionParity(t *testing.T) {
	root := RunGRPCDSL(t, testdata.MessageResultTypeWithViewsDSL)
	services := CreateGRPCServices(root)
	svc := services.Get("ServiceMessageResultTypeWithViews")
	require.NotNil(t, svc)
	endpoint := svc.Endpoint("MethodMessageResultTypeWithViews")
	require.NotNil(t, endpoint)
	require.NotNil(t, endpoint.Method.ViewedResult)
	require.NotNil(t, endpoint.Response.ServerConvert)

	projected := expr.AsObject(endpoint.Method.ViewedResult.Type).Attribute("projected")
	require.NotNil(t, projected)
	projectedName := svc.Service.ViewScope.GoFullTypeName(projected, endpoint.Method.ViewedResult.ViewsPkg)
	projectedRef := svc.Service.ViewScope.GoFullTypeRef(projected, endpoint.Method.ViewedResult.ViewsPkg)

	require.Equal(t, endpoint.Method.ViewedResult.FullRef, endpoint.ViewedResultRef)
	require.Equal(t, projectedName, endpoint.Response.ServerConvert.SrcName)
	require.Equal(t, projectedRef, endpoint.Response.ServerConvert.SrcRef)
	require.Equal(t, []string{"default", "tiny"}, grpcViewNames(endpoint.Method.ViewedResult.Views))
}

func grpcViewNames(views []*service.ViewData) []string {
	names := make([]string, len(views))
	for i, view := range views {
		names[i] = view.Name
	}
	return names
}
