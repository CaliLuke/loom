package openapiimport

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/libopenapi"
	"github.com/stretchr/testify/require"
)

func requireImportedByteSpec(t *testing.T, directory string) {
	t.Helper()
	wire, err := os.ReadFile(filepath.Join(directory, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	document, err := libopenapi.NewDocument(wire)
	require.NoError(t, err)
	_, err = document.BuildV3Model()
	require.NoError(t, err)
	var spec map[string]any
	require.NoError(t, json.Unmarshal(wire, &spec))
	components := requireUnconstrainedMap(t, spec["components"], "components")
	schemas := requireUnconstrainedMap(t, components["schemas"], "schemas")
	payload := requireUnconstrainedMap(t, schemas["Payload"], "payload")
	properties := requireUnconstrainedMap(t, payload["properties"], "properties")
	for _, tc := range []struct {
		name string
		key  string
		want any
	}{
		{"lower", "minLength", float64(4)},
		{"upper", "maxLength", float64(8)},
		{"fallback", "default", "aGk="},
		{"choice", "enum", []any{"aGk="}},
		{"sample", "example", "aGk="},
		{"patterned", "pattern", "^aGk=$"},
	} {
		property := requireUnconstrainedMap(t, properties[tc.name], tc.name)
		require.Equal(t, "string", property["type"], tc.name)
		require.Equal(t, "byte", property["format"], tc.name)
		require.Equal(t, tc.want, property[tc.key], tc.name)
	}
	alias := requireUnconstrainedMap(t, schemas["Encoded"], "encoded alias")
	require.Equal(t, "string", alias["type"])
	require.Equal(t, "byte", alias["format"])
	for name, keyword := range map[string]string{"sequence": "items", "mapping": "additionalProperties"} {
		property := requireUnconstrainedMap(t, properties[name], name)
		child := requireUnconstrainedMap(t, property[keyword], keyword)
		require.Equal(t, "#/components/schemas/Encoded", child["$ref"])
	}
}
