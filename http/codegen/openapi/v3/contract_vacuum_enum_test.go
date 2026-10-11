package openapiv3_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func checkVacuumEnumValues(t *testing.T, vacuum contractVacuum) {
	t.Helper()
	artifacts := renderOpenAPIArtifacts(t, testdata.OpenAPI32FeaturesDSL)
	for _, tc := range []struct {
		name      string
		values    string
		duplicate bool
	}{
		{"distinct arrays", `[[1], [2], [2, 1], [1, 2]]`, false},
		{"duplicate arrays", `[[1], [2], [1]]`, true},
		{"distinct objects", `[{"x": 1}, {"x": 2}, {"y": 1}]`, false},
		{"duplicate objects", `[{"x": 1, "y": 2}, {"y": 2, "x": 1}]`, true},
		{"distinct types", `[1, "1", true, "true", null, "null", "", [], {}]`, false},
		{"duplicate scalars", `[1, 2, 1]`, true},
		{"equivalent numbers", `[1, 1.0, 1e0]`, true},
		{"distinct large numbers", `[9007199254740992, 9007199254740993]`, false},
		{"equivalent large numbers", `[9007199254740993, 9007199254740993.0]`, true},
		{"literal enum properties", `[{"enum": [1, 1]}, {"enum": [2, 2]}]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := decodeOpenAPIJSON(t, artifacts.JSON)
			components := requireMap(t, spec["components"], "components")
			schemas := requireMap(t, components["schemas"], "schemas")
			// Raw JSON preserves number spelling, precision and object member order.
			schemas["EnumProbe"] = map[string]any{"enum": jsontext.Value(tc.values)}
			data, err := json.Marshal(spec, json.Deterministic(true))
			require.NoError(t, err)
			path := filepath.Join(t.TempDir(), "enum.json")
			require.NoError(t, os.WriteFile(path, data, 0o600))
			err = vacuum.lint(path)
			if tc.duplicate {
				require.ErrorContains(t, err, "duplicated-entry-in-enum")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
