package transportir_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/grpc/codegen/internal/transportir"
	grpcdata "github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestBuildServiceCapturesEndpointShapes(t *testing.T) {
	root := expr.RunDSL(t, grpcdata.MessageWithMetadataDSL)
	svc := transportir.BuildService(root.API.GRPC.Services[0])
	require.NotNil(t, svc)
	require.Len(t, svc.Endpoints, 1)

	endpoint := svc.Endpoints[0]
	require.Equal(t, "MethodMessageWithMetadata", endpoint.Name)
	require.Equal(t, "InHeader", endpoint.Response.Headers.KeyName("Location"))
	require.Equal(t, "InTrailer", endpoint.Response.Trailers.KeyName("InTrailer"))
	require.Equal(t, "InMetadata", endpoint.Request.Metadata.KeyName("InMetadata"))
}

func TestBuildServiceCapturesStreamingAndErrors(t *testing.T) {
	root := expr.RunDSL(t, grpcdata.BidirectionalStreamingRPCWithErrorsDSL)
	svc := transportir.BuildService(root.API.GRPC.Services[0])
	require.NotNil(t, svc)
	require.Len(t, svc.Endpoints, 1)

	endpoint := svc.Endpoints[0]
	require.True(t, endpoint.Stream.IsStreaming)
	require.True(t, endpoint.Stream.IsPayloadStreaming)
	require.NotNil(t, endpoint.Request.StreamingMessage)
	require.Len(t, endpoint.Errors, 3)
	require.Equal(t, "timeout", endpoint.Errors[0].Name)
	require.NotNil(t, endpoint.Errors[0].Response.Message)
}
