package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testingx"
)

func TestOpenAPIImportParameterPropertyNamesRoundTrip(t *testing.T) {
	if os.Getenv("LOOM_OPENAPI_CONTRACT") == "" {
		t.Skip("set LOOM_OPENAPI_CONTRACT=1 to run the OpenAPI import contract")
	}
	repoRoot := testingx.RepoRoot()
	loomBin := filepath.Join(t.TempDir(), "loom")
	output, err := testingx.RunCmd(repoRoot, "go", "build", "-o", loomBin, "./cmd/loom")
	require.NoError(t, err, output)
	moduleDir := t.TempDir()
	modulePath := "example.com/parameter-properties"
	writeRoundTripModule(t, moduleDir, modulePath, repoRoot)
	require.NoError(t, os.Mkdir(filepath.Join(moduleDir, "original"), 0o700))
	// This companion fixture has no examples: carrier annotations are outside
	// strict import, and libopenapi #695 also promotes schema examples to headers.
	// IR tests separately check names in examples, including JSON/text variants.
	// https://github.com/pb33f/libopenapi/issues/695
	design := "package original\nimport \"github.com/CaliLuke/loom/http/codegen/testdata\"\nfunc init() { testdata.ParameterPropertyNamesImportDSL() }\n"
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "original", "design.go"), []byte(design), 0o600))
	output, err = testingx.RunCmd(moduleDir, "go", "mod", "tidy")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(moduleDir, loomBin, "gen", modulePath+"/original", "-o", ".")
	require.NoError(t, err, output)
	source, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	source, err = prepareGeneratedOpenAPIContract(source, "parameter-property-names")
	require.NoError(t, err)
	sourceDocument := analyzeOpenAPIContract(t, source)
	input := filepath.Join(moduleDir, "openapi.json")
	require.NoError(t, os.WriteFile(input, source, 0o600))
	output, err = testingx.RunCmd(moduleDir, loomBin, "import", "openapi", input, "-o", "design")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(moduleDir, loomBin, "gen", modulePath+"/design", "-o", ".")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(moduleDir, "go", "mod", "tidy")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(moduleDir, "go", "test", "./...")
	require.NoError(t, err, output)
	regenerated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	regenerated, err = prepareGeneratedOpenAPIContract(regenerated, sourceDocument.Title)
	require.NoError(t, err)
	require.NoError(t, compareOpenAPIContracts(sourceDocument, analyzeOpenAPIContract(t, regenerated)))
}
