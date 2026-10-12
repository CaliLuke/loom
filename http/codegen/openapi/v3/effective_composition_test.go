package openapiv3_test

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestEffectiveCompositionOpenAPI(t *testing.T) {
	artifacts := renderOpenAPIArtifacts(t, testdata.EffectiveCompositionDSL)
	var spec map[string]any
	require.NoError(t, json.Unmarshal(artifacts.JSON, &spec))
	schema := requireComponentSchema(t, spec, "CompositionDerived")
	properties := requireMap(t, schema["properties"], "properties")
	count := requireMap(t, properties["count"], "count")
	require.Equal(t, float64(0), count["minimum"])
	require.Equal(t, float64(100), count["maximum"])
	retries := requireMap(t, properties["retries"], "retries")
	require.Equal(t, float64(5), retries["default"])
	require.ElementsMatch(t, []any{"count", "label"}, schema["required"])
	lintEffectiveCompositionArtifacts(t, artifacts)
}

func TestRecursiveEffectiveCompositionOpenAPI(t *testing.T) {
	artifacts := renderOpenAPIArtifacts(t, testdata.RecursiveEffectiveCompositionDSL)
	spec := decodeOpenAPIJSON(t, artifacts.JSON)
	for _, name := range []string{"RecursiveNode", "ReverseNode"} {
		schema := requireComponentSchema(t, spec, name)
		properties := requireMap(t, schema["properties"], "properties")
		require.Contains(t, properties, "n")
		next := requireMap(t, properties["next"], "next")
		require.Equal(t, "#/components/schemas/"+name, next["$ref"])
	}
	lintEffectiveCompositionArtifacts(t, artifacts)
}

func lintEffectiveCompositionArtifacts(t *testing.T, artifacts renderedOpenAPIArtifacts) {
	t.Helper()
	if os.Getenv("LOOM_OPENAPI_CONTRACT") != "" {
		vacuum := installContractVacuum(t)
		for ext, data := range map[string][]byte{"json": artifacts.JSON, "yaml": artifacts.YAML} {
			path := filepath.Join(t.TempDir(), "openapi."+ext)
			require.NoError(t, os.WriteFile(path, data, 0o600))
			require.NoError(t, vacuum.lint(path))
		}
	}
}
