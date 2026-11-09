package openapiv3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestResultViewRetainsObjectOpenness(t *testing.T) {
	for _, tc := range []struct {
		name            string
		design          func()
		closedAPI       bool
		target, version string
	}{
		{"open-api-32", testdata.ResultViewOpennessDSL, false, "3.2", "3.2.0"},
		{"closed-api-32", testdata.ClosedAPIResultViewOpennessDSL, true, "3.2", "3.2.0"},
		{"open-api-31", testdata.ResultViewOpennessDSL, false, "3.1", "3.1.1"},
		{"closed-api-31", testdata.ClosedAPIResultViewOpennessDSL, true, "3.1", "3.1.1"},
		{"declaration-32", testdata.ResultDeclarationViewOpennessDSL, false, "3.2", "3.2.0"},
		{"declaration-31", testdata.ResultDeclarationViewOpennessDSL, false, "3.1", "3.1.1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := decodeOpenAPIJSON(t, renderOpenAPIArtifactsForVersion(t, tc.design, tc.target, tc.version).JSON)
			resolve := func(schema map[string]any) map[string]any {
				if ref, ok := schema["$ref"].(string); ok {
					return resolveRefMap(t, spec, ref)
				}
				return schema
			}
			response := func(path string) map[string]any {
				operation := requireOperation(t, spec, path, "get")
				responses := requireMap(t, operation["responses"], "responses")
				ok := requireMap(t, responses["200"], "success")
				content := requireMap(t, ok["content"], "response content")
				for _, raw := range content {
					media := requireMap(t, raw, "media")
					return resolve(requireMap(t, media["schema"], "schema"))
				}
				t.Fatal("missing response content")
				return nil
			}
			for _, policy := range []string{"closed", "open", "unset"} {
				for _, shape := range []string{"single", "collection", "nested"} {
					t.Run(policy+"/"+shape, func(t *testing.T) {
						var schema map[string]any
						if shape == "nested" {
							properties := requireMap(t, response("/nested")["properties"], "nested fields")
							schema = resolve(requireMap(t, properties[policy], "nested result"))
						} else {
							schema = response("/" + policy + "/" + shape)
							if shape == "collection" {
								schema = resolve(requireMap(t, schema["items"], "collection result"))
							}
						}
						properties := requireMap(t, schema["properties"], "view fields")
						require.Contains(t, properties, "id")
						require.NotContains(t, properties, "name")
						require.Contains(t, schema["required"], "id")
						if policy == "closed" || policy == "unset" && tc.closedAPI {
							require.Equal(t, false, schema["additionalProperties"])
						} else {
							require.NotContains(t, schema, "additionalProperties")
						}
					})
				}
			}
		})
	}
}
