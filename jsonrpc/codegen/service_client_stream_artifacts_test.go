package codegen

import (
	"os"
	"path/filepath"
	"testing"

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
		{"jsonrpcSSEMixedResultsDSL", jsonrpcSSEMixedResultsDSL},
		{"jsonrpcWebSocketStreamingPayloadDSL", jsonrpcWebSocketStreamingPayloadDSL},
		{"jsonrpcWebSocketPendingOrderDSL", jsonrpcWebSocketPendingOrderDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(output, "jsonrpc", tc.name)
			require.NoError(t, os.MkdirAll(dir, 0o700))
			renderJSONRPCModule(t, dir, "example.com/streamcontract", RunJSONRPCDSL(t, tc.design))
		})
	}
}
