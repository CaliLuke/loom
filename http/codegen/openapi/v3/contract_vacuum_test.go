package openapiv3_test

import (
	"crypto/sha256"
	"debug/buildinfo"
	"embed"
	"encoding/json/v2"
	"errors"
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
	adapter string
	ruleset string
}

type contractDocument struct {
	URI, Path string
}

type contractVacuumReport struct {
	BinarySHA256 string
	Rules        []string
	Findings     []struct {
		Rule, Severity, Path, Message string
		Line, Column                  int
	}
	Resources []struct {
		Role, URI, ContentSHA256, IndexedContentSHA256 string
	}
	Errors []string
}

type contractVacuumInvalidCase struct {
	name       string
	mutate     func(*testing.T, map[string]any)
	diagnostic string
}

// Required unreleased fix for external security requirement URIs:
// https://github.com/daveshanley/vacuum/issues/1056 and PR1029.
// Return to a stable release containing c8d7f10d3472db624b17892eebf679471c922c47
// once the existing positive and negative contract controls pass unchanged.
const vacuumVersion = "v0.39.4-0.20261107034229-b6c4e203d2ea"

//go:embed testdata/vacuum_adapter/*.go.txt
var contractVacuumAdapter embed.FS

func installContractVacuum(t *testing.T) contractVacuum {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GOBIN", dir)
	libraryVersion, err := testingx.RunCmd(".", "go", "list", "-m", "-f", "{{.Version}}", "github.com/pb33f/libopenapi")
	require.NoError(t, err)
	libraryVersion = strings.TrimSpace(libraryVersion)
	require.NotEmpty(t, libraryVersion)
	ruleset, err := filepath.Abs("testdata/vacuum.yaml")
	require.NoError(t, err)
	moduleDir := filepath.Join(dir, "adapter")
	require.NoError(t, os.Mkdir(moduleDir, 0o700))
	module := fmt.Sprintf("module example.com/loom-contract-vacuum\n\ngo 1.27.3\n\nrequire (\n\tgithub.com/daveshanley/vacuum %s\n\tgithub.com/pb33f/libopenapi %s\n)\n", vacuumVersion, libraryVersion)
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(module), 0o600))
	files, err := contractVacuumAdapter.ReadDir("testdata/vacuum_adapter")
	require.NoError(t, err)
	for _, file := range files {
		data, err := contractVacuumAdapter.ReadFile("testdata/vacuum_adapter/" + file.Name())
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(moduleDir, strings.TrimSuffix(file.Name(), ".txt")), data, 0o600))
	}
	_, err = testingx.RunCmd(moduleDir, "go", "mod", "tidy")
	require.NoError(t, err)
	_, err = testingx.RunCmd(moduleDir, "go", "install", "-mod=mod", "github.com/daveshanley/vacuum")
	require.NoError(t, err)
	adapter := filepath.Join(dir, "vacuum-resources")
	_, err = testingx.RunCmd(moduleDir, "go", "build", "-o", adapter, ".")
	require.NoError(t, err)
	for _, binary := range []string{filepath.Join(dir, "vacuum"), adapter} {
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
		data, err := os.ReadFile(binary)
		require.NoError(t, err)
		t.Logf("built %s sha256=%x", filepath.Base(binary), sha256.Sum256(data))
	}
	return contractVacuum{binary: filepath.Join(dir, "vacuum"), adapter: adapter, ruleset: ruleset}
}

func checkVacuumInvalidContracts(t *testing.T, vacuum contractVacuum) {
	t.Helper()
	artifacts := renderOpenAPIArtifacts(t, testdata.OpenAPI32FeaturesDSL)
	for _, tc := range vacuumInvalidContracts() {
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

func (v contractVacuum) lint(path string, documents ...contractDocument) error {
	if len(documents) > 0 {
		_, err := v.motor(path, documents...)
		return err
	}
	_, err := testingx.RunCmd(filepath.Dir(path), v.binary, "lint", "--no-update-check",
		"--no-banner", "--no-style", "--details", "--errors", "--no-clip",
		"--fail-severity", "error", "--ruleset", v.ruleset, path)
	return err
}

func (v contractVacuum) motor(path string, documents ...contractDocument) (contractVacuumReport, error) {
	args := make([]string, 0, 2+2*len(documents))
	args = append(args, v.ruleset, path)
	for _, document := range documents {
		args = append(args, document.URI, document.Path)
	}
	output, runErr := testingx.RunCmd(filepath.Dir(path), v.adapter, args...)
	var report contractVacuumReport
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		return report, errors.Join(runErr, fmt.Errorf("decode Vacuum resource report: %w", err))
	}
	if runErr != nil {
		return report, fmt.Errorf("Vacuum resource lint: %w; report=%s", runErr, output)
	}
	return report, nil
}

func vacuumInvalidContracts() []contractVacuumInvalidCase {
	return []contractVacuumInvalidCase{
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
	}
}
