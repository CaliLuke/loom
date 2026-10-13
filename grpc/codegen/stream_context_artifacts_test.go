package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestStreamContextArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_STREAM_CONTEXT_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_STREAM_CONTEXT_OUTPUT for bounded artifact comparison")
	}
	source := os.Getenv("LOOM_DIR")
	require.NotEmpty(t, source, "artifact comparison requires an explicit Loom source")
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"server", testdata.ServerStreamingRPCDSL},
		{"client", testdata.ClientStreamingRPCDSL},
		{"client-empty", testdata.ClientStreamingNoResultDSL},
		{"duplex", testdata.BidirectionalStreamingRPCDSL},
		{"shared-sse", serviceClientSharedSSEDSL},
		{"context", grpcMessageContextDSL},
		{"admission", testdata.StreamAdmissionDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(output, tc.name)
			require.NoError(t, os.MkdirAll(dir, 0o700))
			renderGRPCModule(t, dir, "example.com/streamcontext", RunGRPCDSL(t, tc.design), source)
		})
	}
}
