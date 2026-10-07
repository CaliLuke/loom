package openapiv3_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestUntaggedArrayObjectContract(t *testing.T) {
	for _, version := range []struct{ target, want string }{
		{"3.1", openapiv3.OpenAPICompatibilityVersion}, {"3.2", openapiv3.OpenAPIVersion},
	} {
		t.Run(version.target, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, testdata.UntaggedArraysDSL, version.target, version.want)
			spec := decodeOpenAPIJSON(t, artifacts.JSON)
			schemas := spec["components"].(map[string]any)["schemas"].(map[string]any)
			found := false
			for _, raw := range schemas {
				schema := raw.(map[string]any)
				branches, ok := schema["oneOf"].([]any)
				if !ok || len(branches) != 2 {
					continue
				}
				kinds := make(map[string]bool)
				for _, branch := range branches {
					ref := branch.(map[string]any)["$ref"].(string)
					component := schemas[filepath.Base(ref)].(map[string]any)
					kind, ok := component["type"].(string)
					if ok {
						kinds[kind] = true
					}
				}
				if kinds["array"] && kinds["object"] {
					found = true
					require.NotContains(t, schema, "discriminator")
				}
			}
			require.True(t, found, "missing mixed array/object oneOf")
			if os.Getenv("LOOM_OPENAPI_CONTRACT") != "" {
				dir := t.TempDir()
				path := filepath.Join(dir, "openapi.yaml")
				require.NoError(t, os.WriteFile(path, artifacts.YAML, 0o600))
				_, err := testingx.RunCmd(dir, "npx", "--yes", "@redocly/cli@"+redoclyCLIVersion, "lint", path)
				require.NoError(t, err)
			}
		})
	}
}
