package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	servicedata "github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestBasicCLISourcesGenerated(t *testing.T) {
	for _, mapping := range []string{"basic", "query-header", "optional", "defaulted", "repeated", "ordinary-repeated", "scalar-body", "object-body"} {
		t.Run(mapping, func(t *testing.T) {
			dir := t.TempDir()
			renderBasicCLISources(t, dir, mapping)
			harness := strings.NewReplacer("TRANSPORT", "jsonrpc", "MAPPING", mapping).Replace(servicedata.BasicCLISourcesHarness)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "sources_test.go"), []byte(harness), 0o600))
			runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
			runGoJSONRPCTestCommand(t, dir, "build", "./...")
			runGoJSONRPCTestCommand(t, dir, "vet", "./...")
			runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
		})
	}
}

func TestBasicCLISourceArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_BASIC_SOURCES_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_BASIC_SOURCES_OUTPUT for isolated artifact comparison")
	}
	for _, mapping := range []string{"basic", "query-header", "optional", "defaulted", "repeated", "ordinary-repeated", "scalar-body", "object-body"} {
		t.Run(mapping, func(t *testing.T) {
			dir := filepath.Join(output, "jsonrpc", mapping)
			require.NoError(t, os.MkdirAll(dir, 0o700))
			renderBasicCLISources(t, dir, mapping)
		})
	}
}

func renderBasicCLISources(t *testing.T, dir, mapping string) {
	t.Helper()
	root := RunJSONRPCDSL(t, servicedata.BasicCLISourcesDSL("jsonrpc", mapping))
	renderJSONRPCModule(t, dir, "example.com/sources", root)
	renderCodegenFiles(t, dir, ClientCLIFiles("example.com/sources/gen", CreateJSONRPCServices(root)))
	path := filepath.Join(dir, "go.mod")
	module, err := os.ReadFile(path)
	require.NoError(t, err)
	module = []byte(strings.Replace(string(module), "github.com/CaliLuke/loom v1.0.0", "github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261013020843-063198a2f427", 1))
	require.NoError(t, os.WriteFile(path, module, 0o600))
}
