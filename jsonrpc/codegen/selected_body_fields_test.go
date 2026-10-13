package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	servicedata "github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestSelectedBodyFieldIdentityCompiles(t *testing.T) {
	for _, direction := range []string{"request", "response", "error", "whole", "viewed"} {
		for _, renamed := range []bool{false, true} {
			name := direction + "/ordinary"
			if renamed {
				name = direction + "/renamed"
			}
			t.Run(name, func(t *testing.T) {
				dir := t.TempDir()
				renderSelectedBodyFields(t, dir, direction, "string", renamed)
				if direction == "request" || direction == "response" {
					require.NoError(t, os.WriteFile(filepath.Join(dir, "wire_test.go"), []byte(servicedata.SelectedBodyFieldHarness("jsonrpc", direction, "string", renamed)), 0o600))
				}
				runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
				runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
				runGoJSONRPCTestCommand(t, dir, "vet", "./...")
			})
		}
	}
}

func TestSelectedBodyFieldCarriersCompile(t *testing.T) {
	for _, shape := range []string{"named", "optional", "defaulted", "nullable", "array", "object"} {
		t.Run(shape, func(t *testing.T) {
			dir := t.TempDir()
			renderSelectedBodyFields(t, dir, "request", shape, true)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "wire_test.go"), []byte(servicedata.SelectedBodyFieldHarness("jsonrpc", "request", shape, true)), 0o600))
			runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
			runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
			runGoJSONRPCTestCommand(t, dir, "vet", "./...")
		})
	}
}

func TestSelectedBodyFieldArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_SELECTED_BODY_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_SELECTED_BODY_OUTPUT for isolated artifact comparison")
	}
	for _, direction := range []string{"request", "response", "error", "whole", "viewed"} {
		for _, renamed := range []bool{false, true} {
			name := direction + "-ordinary"
			if renamed {
				name = direction + "-renamed"
			}
			renderSelectedBodyFields(t, filepath.Join(output, "jsonrpc", name), direction, "string", renamed)
		}
	}
	for _, shape := range []string{"named", "optional", "defaulted", "nullable", "array", "object"} {
		renderSelectedBodyFields(t, filepath.Join(output, "jsonrpc", shape), "request", shape, true)
	}
}

func renderSelectedBodyFields(t *testing.T, dir, direction, shape string, renamed bool) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o700))
	root := RunJSONRPCDSL(t, servicedata.SelectedBodyFieldsDSL("jsonrpc", direction, shape, renamed))
	renderJSONRPCModule(t, dir, "example.com/bodyfields", root)
	renderCodegenFiles(t, dir, ClientCLIFiles("example.com/bodyfields/gen", CreateJSONRPCServices(root)))
	path := filepath.Join(dir, "go.mod")
	module, err := os.ReadFile(path)
	require.NoError(t, err)
	module = []byte(strings.Replace(string(module), "github.com/CaliLuke/loom v1.0.0", "github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261013020843-063198a2f427", 1))
	require.NoError(t, os.WriteFile(path, module, 0o600))
}

func TestSelectedOptionalViewedBodyCompiles(t *testing.T) {
	for _, renamed := range []bool{false, true} {
		dir := t.TempDir()
		renderSelectedBodyFields(t, dir, "viewed", "optional", renamed)
		runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
		runGoJSONRPCTestCommand(t, dir, "build", "./...")
		runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	}
}
