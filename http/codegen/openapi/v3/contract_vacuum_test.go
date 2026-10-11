package openapiv3_test

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

type contractVacuum struct {
	binary  string
	ruleset string
}

const (
	vacuumVersion = "v0.33.0"
	// Remove this override when a released Vacuum includes the fix in
	// https://github.com/pb33f/libopenapi/pull/656 and passes the enum controls.
	vacuumLibopenAPIVersion = "v0.41.7-0.20261011180705-d696afb9dc03"
)

func installContractVacuum(t *testing.T) contractVacuum {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GOBIN", dir)
	_, err := testingx.RunCmd(dir, "go", "mod", "init", "loom-openapi-contract")
	require.NoError(t, err)
	_, err = testingx.RunCmd(dir, "go", "mod", "edit",
		"-require=github.com/daveshanley/vacuum@"+vacuumVersion,
		"-replace=github.com/pb33f/libopenapi=github.com/CaliLuke/libopenapi@"+vacuumLibopenAPIVersion)
	require.NoError(t, err)
	// Installing without @version honors the isolated module's dependency pin.
	_, err = testingx.RunCmd(dir, "go", "install", "-mod=mod", "github.com/daveshanley/vacuum")
	require.NoError(t, err)
	ruleset, err := filepath.Abs("testdata/vacuum.yaml")
	require.NoError(t, err)
	return contractVacuum{binary: filepath.Join(dir, "vacuum"), ruleset: ruleset}
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
