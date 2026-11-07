package openapiv3_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestUntaggedUnionBothTargets(t *testing.T) {
	for _, tc := range []struct {
		name, target, version string
		design                func()
	}{
		{"forms", "3.2", "3.2.0", testdata.UntaggedFormDSL},
		{"forms", "3.1", "3.1.1", testdata.UntaggedFormDSL},
		{"roots", "3.2", "3.2.0", testdata.UntaggedRootsDSL},
		{"roots", "3.1", "3.1.1", testdata.UntaggedRootsDSL},
		{"locations", "3.2", "3.2.0", testdata.UntaggedLocationsDSL},
		{"locations", "3.1", "3.1.1", testdata.UntaggedLocations31DSL},
	} {
		t.Run(tc.name+"/"+tc.target, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, tc.design, tc.target, tc.version)
			validateOpenAPIVersion(t, artifacts.JSON, tc.version)
			validateOpenAPIVersion(t, artifacts.YAML, tc.version)
			if tc.name == "roots" {
				spec := decodeOpenAPIJSON(t, artifacts.JSON)
				components := requireMap(t, spec["components"], "components")
				resolve := func(value map[string]any, kind string) map[string]any {
					if ref, ok := value["$ref"].(string); ok {
						return requireMap(t, requireMap(t, components[kind], kind)[strings.TrimPrefix(ref, "#/components/"+kind+"/")], ref)
					}
					return value
				}
				for _, path := range []string{"/openbody", "/closedbody"} {
					operation := requireOperation(t, spec, path, "post")
					body := resolve(requireMap(t, operation["requestBody"], "body"), "requestBodies")
					media := requireMap(t, requireMap(t, body["content"], "content")["application/json"], "media")
					schema := resolve(requireMap(t, media["schema"], "body schema"), "schemas")
					value := resolve(requireMap(t, requireMap(t, schema["properties"], "properties")["value"], "value"), "schemas")
					if path == "/closedbody" {
						require.Equal(t, false, value["unevaluatedProperties"])
					} else {
						require.NotContains(t, value, "unevaluatedProperties")
					}
				}
			}
		})
	}
}
