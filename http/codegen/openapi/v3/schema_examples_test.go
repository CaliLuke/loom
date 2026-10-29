package openapiv3_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRenderedMultipleSchemaExamples(t *testing.T) {
	var vacuum contractVacuum
	if os.Getenv("LOOM_OPENAPI_CONTRACT") != "" {
		vacuum = installContractVacuum(t)
	}
	for _, version := range []struct{ target, version string }{{"3.1", "3.1.1"}, {"3.2", "3.2.0"}} {
		t.Run(version.target, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, testdata.MultipleSchemaExamplesDSL, version.target, version.version)
			for _, spec := range []map[string]any{decodeOpenAPIJSON(t, artifacts.JSON), decodeOpenAPIExampleYAML(t, artifacts.YAML)} {
				components := requireMap(t, spec["components"], "components")
				schemas := requireMap(t, components["schemas"], "schemas")
				operation := requireOperation(t, spec, "/schema-examples", "post")
				media := requireResponseMediaType(t, operation, "application/json")
				require.Equal(t, "green", media["example"])
				colors := requireMap(t, media["schema"], "colors")
				if ref, ok := colors["$ref"].(string); ok && colors["examples"] == nil {
					colors = requireMap(t, schemas[strings.TrimPrefix(ref, "#/components/schemas/")], "referenced colors")
				}
				require.Equal(t, []any{nil, "red", "blue", "blue"}, colors["examples"])
				require.NotContains(t, colors, "example")
			}
			if vacuum.binary != "" {
				for ext, data := range map[string][]byte{"json": artifacts.JSON, "yaml": artifacts.YAML} {
					path := filepath.Join(t.TempDir(), "openapi."+ext)
					require.NoError(t, os.WriteFile(path, data, 0o600))
					require.NoError(t, vacuum.lint(path))
				}
			}
		})
	}
}
