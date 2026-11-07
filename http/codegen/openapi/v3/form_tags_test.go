package openapiv3_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestCapturedFormTagsExport(t *testing.T) {
	for _, version := range []struct{ target, version string }{{"3.1", "3.1.1"}, {"3.2", "3.2.0"}} {
		t.Run(version.target, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, testdata.UntaggedFormDSL, version.target, version.version)
			validateOpenAPIVersion(t, artifacts.JSON, version.version)
			validateOpenAPIVersion(t, artifacts.YAML, version.version)
			spec := decodeOpenAPIJSON(t, artifacts.JSON)
			schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
			operation := requireOperation(t, spec, "/tags", "post")
			body := requireMap(t, operation["requestBody"], "body")
			media := requireMap(t, requireMap(t, body["content"], "content")["application/x-www-form-urlencoded"], "form")
			schema := requireMap(t, media["schema"], "schema")
			if ref, ok := schema["$ref"].(string); ok {
				schema = requireMap(t, schemas[strings.TrimPrefix(ref, "#/components/schemas/")], "component")
			}
			properties := requireMap(t, schema["properties"], "properties")
			require.Len(t, properties, 5)
			require.ElementsMatch(t, []any{"jsonWire", "wire", "selection", "nums", "default"}, schema["required"])
			for _, name := range []string{"jsonWire", "wire", "selection", "nums", "default"} {
				require.Contains(t, properties, name)
			}
			encoding := requireMap(t, media["encoding"], "encoding")
			require.Equal(t, "application/json", requireMap(t, encoding["jsonWire"], "JSON field")["contentType"])
			require.Equal(t, "pipeDelimited", requireMap(t, encoding["nums"], "styled field")["style"])
			require.NotContains(t, encoding, "value")
			require.NotContains(t, encoding, "numbers")
			exploded := requireOperation(t, spec, "/exploded", "post")
			explodedBody := requireMap(t, exploded["requestBody"], "exploded body")
			explodedMedia := requireMap(t, requireMap(t, explodedBody["content"], "content")["application/x-www-form-urlencoded"], "exploded form")
			explodedEncoding := requireMap(t, explodedMedia["encoding"], "exploded encoding")
			require.Equal(t, "form", requireMap(t, explodedEncoding["value"], "union field")["style"])
		})
	}
}
