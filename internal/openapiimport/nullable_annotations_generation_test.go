package openapiimport

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestNullableAnnotationsGeneratedContract(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		name := "ordinary"
		if nullable {
			name = "nullable"
		}
		t.Run(name, func(t *testing.T) {
			var source map[string]any
			require.NoError(t, json.Unmarshal(readFixture(t, "supported.json"), &source))
			properties := source["components"].(map[string]any)["schemas"].(map[string]any)["Pet"].(map[string]any)["properties"].(map[string]any)
			if nullable {
				for _, field := range []string{"id", "secret", "nickname"} {
					branch := properties[field].(map[string]any)
					outer := map[string]any{"title": "Nullable " + field}
					for _, key := range []string{"description", "deprecated", "readOnly", "writeOnly"} {
						if value, ok := branch[key]; ok {
							outer[key] = value
							delete(branch, key)
						}
					}
					outer["anyOf"] = []any{branch, map[string]any{"type": "null"}}
					properties[field] = outer
				}
			}
			encoded, err := json.Marshal(source, json.Deterministic(true))
			require.NoError(t, err)
			document, diagnostics, err := Analyze(encoded)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			moduleDir := requireRenderedDesignGenerates(t, rendered)
			wire, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
			require.NoError(t, err)
			parsed, err := libopenapi.NewDocument(wire)
			require.NoError(t, err)
			_, err = parsed.BuildV3Model()
			require.NoError(t, err)
			if evidence := os.Getenv("LOOM_ANNOTATION_EVIDENCE"); evidence != "" {
				output := filepath.Join(evidence, name)
				require.NoError(t, os.MkdirAll(output, 0750))
				require.NoError(t, os.WriteFile(filepath.Join(output, "design.go"), rendered, 0600))
				require.NoError(t, os.CopyFS(filepath.Join(output, "gen"), os.DirFS(filepath.Join(moduleDir, "gen"))))
			}
			contract := readGeneratedOpenAPIContract(t, moduleDir)
			got := contract["components"].(map[string]any)["schemas"].(map[string]any)["Pet"].(map[string]any)["properties"].(map[string]any)
			require.Equal(t, true, got["id"].(map[string]any)["readOnly"])
			require.Equal(t, true, got["secret"].(map[string]any)["writeOnly"])
			require.Equal(t, true, got["nickname"].(map[string]any)["deprecated"])
			if nullable {
				for _, field := range []string{"id", "secret", "nickname"} {
					require.Equal(t, "Nullable "+field, got[field].(map[string]any)["title"])
				}
			}
		})
	}
}
