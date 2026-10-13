package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/http/codegen/testdata"
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
		{"StreamingResultDSL", testdata.StreamingResultDSL},
		{"StreamingPayloadDSL", testdata.StreamingPayloadDSL},
		{"StreamingPayloadNoResultDSL", testdata.StreamingPayloadNoResultDSL},
		{"BidirectionalStreamingDSL", testdata.BidirectionalStreamingDSL},
		{"MixedResultsDSL", testdata.MixedResultsDSL},
		{"SSEResultTypesDSL", testdata.SSEResultTypesDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(output, "http", tc.name)
			require.NoError(t, os.MkdirAll(dir, 0o700))
			renderHTTPModule(t, dir, "example.com/streamcontract", RunHTTPDSL(t, tc.design))
		})
	}
}
