package openapiimport

import (
	"encoding/json/v2"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
	"go.yaml.in/yaml/v3"
)

// These specimens use only the common type/null vocabulary supported by Draft 7
// and 2020-12. This checks instances, not full OpenAPI or 3.2 conformance.
func requireTypeNullAcceptance(t *testing.T, source []byte, moduleDir, component string, cases []struct {
	value any
	valid bool
}) {
	t.Helper()
	var original map[string]any
	require.NoError(t, yaml.Unmarshal(source, &original))
	generated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	var regenerated map[string]any
	require.NoError(t, json.Unmarshal(generated, &regenerated))
	for _, doc := range []map[string]any{original, regenerated} {
		compiled, err := gojsonschema.NewSchema(gojsonschema.NewGoLoader(map[string]any{
			"$ref":       "#/components/schemas/" + component,
			"components": doc["components"],
		}))
		require.NoError(t, err)
		for _, tc := range cases {
			result, err := compiled.Validate(gojsonschema.NewGoLoader(tc.value))
			require.NoError(t, err)
			require.Equal(t, tc.valid, result.Valid(), "component %s value %#v: %v", component, tc.value, result.Errors())
		}
	}
}

func requireIndependentRegeneration(t *testing.T, moduleDir string) {
	t.Helper()
	before := make(map[string][]byte)
	for _, name := range []string{"openapi.json", "openapi.yaml"} {
		content, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", name))
		require.NoError(t, err)
		before[name] = content
	}
	command := exec.Command(filepath.Join(moduleDir, "loom"), "gen", "example.com/imported/design", "-o", ".")
	command.Dir = moduleDir
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	for name, content := range before {
		after, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", name))
		require.NoError(t, err)
		require.Equal(t, content, after, "independent process regenerated %s", name)
	}
}
