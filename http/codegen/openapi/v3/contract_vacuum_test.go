package openapiv3_test

import (
	"debug/buildinfo"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

type contractVacuum struct {
	binary  string
	ruleset string
}

// Required unreleased fix for external security requirement URIs:
// https://github.com/daveshanley/vacuum/issues/1056 and PR1029.
// Return to a stable release containing c8d7f10d3472db624b17892eebf679471c922c47
// once the existing positive and negative contract controls pass unchanged.
const vacuumVersion = "v0.39.4-0.20261107034229-b6c4e203d2ea"

func installContractVacuum(t *testing.T) contractVacuum {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GOBIN", dir)
	libraryVersion, err := testingx.RunCmd(".", "go", "list", "-m", "-f", "{{.Version}}", "github.com/pb33f/libopenapi")
	require.NoError(t, err)
	libraryVersion = strings.TrimSpace(libraryVersion)
	require.NotEmpty(t, libraryVersion)
	module := fmt.Sprintf("module example.com/loom-contract-vacuum\n\ngo 1.27.3\n\nrequire (\n\tgithub.com/daveshanley/vacuum %s\n\tgithub.com/pb33f/libopenapi %s\n)\n", vacuumVersion, libraryVersion)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0o600))
	_, err = testingx.RunCmd(dir, "go", "install", "-mod=mod", "github.com/daveshanley/vacuum")
	require.NoError(t, err)
	binary := filepath.Join(dir, "vacuum")
	build, err := buildinfo.ReadFile(binary)
	require.NoError(t, err)
	var builtLibraryVersion string
	for _, dependency := range build.Deps {
		if dependency.Path == "github.com/pb33f/libopenapi" {
			builtLibraryVersion = dependency.Version
			break
		}
	}
	require.Equal(t, libraryVersion, builtLibraryVersion, "Vacuum must use the framework's selected parser version")
	ruleset, err := filepath.Abs("testdata/vacuum.yaml")
	require.NoError(t, err)
	return contractVacuum{binary: binary, ruleset: ruleset}
}

func checkVacuumInvalidContracts(t *testing.T, vacuum contractVacuum) {
	t.Helper()
	artifacts := renderOpenAPIArtifacts(t, testdata.OpenAPI32FeaturesDSL)
	for _, tc := range []struct {
		name       string
		mutate     func(*testing.T, map[string]any)
		diagnostic string
	}{
		{"missing info", func(_ *testing.T, spec map[string]any) {
			delete(spec, "info")
		}, "oas3-schema"},
		{"parameter location", func(t *testing.T, spec map[string]any) {
			operation := requireOperation(t, spec, "/books", "query")
			parameters := requireSlice(t, operation["parameters"], "parameters")
			requireMap(t, parameters[0], "parameter")["in"] = "invalid"
		}, "oas3-schema"},
		{"schema type", func(t *testing.T, spec map[string]any) {
			requireComponentSchema(t, spec, "CatalogCreated")["type"] = "invalid"
		}, "oas-schema-check"},
		{"device authorization URL", func(t *testing.T, spec map[string]any) {
			components := requireMap(t, spec["components"], "components")
			schemes := requireMap(t, components["securitySchemes"], "schemes")
			flows := requireMap(t, requireMap(t, schemes["device"], "device")["flows"], "flows")
			delete(requireMap(t, flows["deviceAuthorization"], "device flow"), "deviceAuthorizationUrl")
		}, "oas3-schema"},
		{"conflicting encodings", func(t *testing.T, spec map[string]any) {
			components := requireMap(t, spec["components"], "components")
			mediaTypes := requireMap(t, components["mediaTypes"], "media types")
			stream := requireMap(t, mediaTypes["CatalogEventStream"], "stream")
			requireMap(t, stream["itemEncoding"], "item encoding")["encoding"] = map[string]any{
				"id": map[string]any{"style": "form"},
			}
		}, "oas3-schema"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := decodeOpenAPIJSON(t, artifacts.JSON)
			tc.mutate(t, spec)
			data, err := json.Marshal(spec, json.Deterministic(true))
			require.NoError(t, err)
			path := filepath.Join(t.TempDir(), "invalid.json")
			require.NoError(t, os.WriteFile(path, data, 0o600))
			require.ErrorContains(t, vacuum.lint(path), tc.diagnostic)
		})
	}
}

func (v contractVacuum) lint(path string) error {
	_, err := testingx.RunCmd(filepath.Dir(path), v.binary, "lint", "--no-update-check",
		"--no-banner", "--no-style", "--details", "--errors", "--no-clip",
		"--fail-severity", "error", "--ruleset", v.ruleset, path)
	return err
}
