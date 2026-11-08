package openapiimport

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestImportSSEArtifacts retains the same required inputs used by the compiled
// DSL tests for independent-process output comparison. It does not generate a
// transport or claim an exporter/importer roundtrip.
func TestImportSSEArtifacts(t *testing.T) {
	directory := os.Getenv("LOOM_IMPORT_SSE_OUTPUT")
	if directory == "" {
		t.Skip("set LOOM_IMPORT_SSE_OUTPUT to retain the bounded importer outputs")
	}
	inputs := map[string][]byte{"ordinary": readFixture(t, "supported.yaml")}
	if os.Getenv("LOOM_IMPORT_SSE_BASE_ONLY") == "" {
		inputs["class"] = sseInverseSource([]string{"2XX"}, "", "")
		inputs["created_header"] = sseInverseSource([]string{"201"}, "", "")
		inputs["shared_exact"] = sseInverseSource([]string{"200", "201"}, "", "")
		inputs["comparison_map"] = sseComparisonMapSource(sseAsyncContract("['201']", sseComparisonMapExpansion))
		inputs["producer_metadata"] = producerSSEMetadataSource()
		inputs["producer_primitive"] = readFixture(t, "producer_primitive.json")
		inputs["producer_server"] = producerServersSource("3.2.0", "  - {name: 'test api', url: 'http://localhost:80', description: 'Default server for test api'}\n")
	}
	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			document, diagnostics, err := Analyze(input)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			source, err := Render(document, Options{})
			require.NoError(t, err)
			require.NoError(t, os.MkdirAll(directory, 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(directory, name+".yaml"), input, 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(directory, name+".go"), source, 0o600))
		})
	}
}
