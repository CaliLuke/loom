package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/grpc/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestServiceClientStreamArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_CLIENT_STREAM_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_CLIENT_STREAM_OUTPUT for bounded artifact comparison")
	}
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"ServerStreamingRPCDSL", testdata.ServerStreamingRPCDSL},
		{"ClientStreamingRPCDSL", testdata.ClientStreamingRPCDSL},
		{"ClientStreamingNoResultDSL", testdata.ClientStreamingNoResultDSL},
		{"BidirectionalStreamingRPCDSL", testdata.BidirectionalStreamingRPCDSL},
		{"serviceClientSharedSSEDSL", serviceClientSharedSSEDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(output, "grpc", tc.name)
			require.NoError(t, os.MkdirAll(dir, 0o700))
			renderGRPCResponseContractModule(t, dir, "example.com/streamcontract", RunGRPCDSL(t, tc.design))
		})
	}
}
