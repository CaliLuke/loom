package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	servicedata "github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestSelectedBodyContractsGenerated compiles all body directions and carriers
// together while retaining the client/server and CLI assertions for each case.
func TestSelectedBodyContractsGenerated(t *testing.T) {
	type bodyCase struct {
		name, direction, shape string
		renamed                bool
	}
	cases := make([]bodyCase, 0, 18)
	for _, direction := range []string{"request", "response", "error", "whole", "viewed"} {
		for _, renamed := range []bool{false, true} {
			name := direction + "/ordinary"
			if renamed {
				name = direction + "/renamed"
			}
			cases = append(cases, bodyCase{name, direction, "string", renamed})
		}
	}
	for _, shape := range []string{"named", "optional", "defaulted", "nullable", "array", "object"} {
		cases = append(cases, bodyCase{"request/" + shape, "request", shape, true})
	}
	for _, renamed := range []bool{false, true} {
		name := "viewed/optional/ordinary"
		if renamed {
			name = "viewed/optional/renamed"
		}
		cases = append(cases, bodyCase{name, "viewed", "optional", renamed})
	}
	batch := testingx.NewModuleBatch(t, resolveJSONRPCTestLoomModulePath(t, t.TempDir()), "")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := RunJSONRPCDSL(t, servicedata.SelectedBodyFieldsDSL("jsonrpc", tc.direction, tc.shape, tc.renamed))
			c := batch.Case(t, tc.name, "example.com/bodyfields")
			renderJSONRPCModuleFiles(t, c.Dir, c.ModulePath+"/gen", root)
			renderCodegenFiles(t, c.Dir, ClientCLIFiles(c.ModulePath+"/gen", CreateJSONRPCServices(root)))
			if tc.direction == "request" || tc.direction == "response" {
				c.WriteFile(t, "wire_test.go", servicedata.SelectedBodyFieldHarness("jsonrpc", tc.direction, tc.shape, tc.renamed))
			}
		})
	}
	batch.Check(t)
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
