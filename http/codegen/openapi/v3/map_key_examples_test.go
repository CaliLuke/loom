package openapiv3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestFilesRenderBoolKeyedQueryMapExamplesWithStringKeys(t *testing.T) {
	cases := []struct {
		name string
		dsl  func()
	}{
		{"map-bool-bool", testdata.PayloadQueryMapBoolBoolDSL},
		{"map-bool-bool-validate", testdata.PayloadQueryMapBoolBoolValidateDSL},
		{"map-bool-string", testdata.PayloadQueryMapBoolStringDSL},
		{"map-bool-string-validate", testdata.PayloadQueryMapBoolStringValidateDSL},
		{"map-bool-array-bool", testdata.PayloadQueryMapBoolArrayBoolDSL},
		{"map-bool-array-bool-validate", testdata.PayloadQueryMapBoolArrayBoolValidateDSL},
		{"map-bool-array-string", testdata.PayloadQueryMapBoolArrayStringDSL},
		{"map-bool-array-string-validate", testdata.PayloadQueryMapBoolArrayStringValidateDSL},
		{"primitive-map-bool-array-bool-validate", testdata.PayloadQueryPrimitiveMapBoolArrayBoolValidateDSL},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			artifacts := renderOpenAPIArtifacts(t, tc.dsl)
			for format, spec := range map[string]map[string]any{
				"json": decodeOpenAPIJSON(t, artifacts.JSON),
				"yaml": decodeOpenAPIExampleYAML(t, artifacts.YAML),
			} {
				parameter := parameterByName(t, requireOperation(t, spec, "/", "get"), "q")
				require.Equal(t, "query", parameter["in"], format)
				schema := requireMap(t, parameter["schema"], format+" q schema")
				for label, example := range map[string]any{
					"parameter example": parameter["example"],
					"schema example":    schema["example"],
				} {
					if example == nil {
						continue
					}
					members := requireMap(t, example, format+" "+label)
					require.NotEmpty(t, members, format+" "+label)
					for key := range members {
						require.Contains(t, []string{"true", "false"}, key, format+" "+label)
					}
				}
			}
		})
	}
}

func TestFilesRenderScalarMapKeysAsJSONMemberNames(t *testing.T) {
	artifacts := renderOpenAPIArtifacts(t, testdata.OpenAPIScalarMapKeysDSL)
	for format, spec := range map[string]map[string]any{
		"json": decodeOpenAPIJSON(t, artifacts.JSON),
		"yaml": decodeOpenAPIExampleYAML(t, artifacts.YAML),
	} {
		list := requireOperation(t, spec, "/flags", "get")
		enabled := parameterByName(t, list, "enabled")
		for key, value := range requireMap(t, enabled["example"], format+" enabled example") {
			require.Contains(t, []string{"true", "false"}, key, format)
			require.NotEmpty(t, requireSlice(t, value, format+" enabled example value"), format)
		}
		labels := parameterByName(t, list, "labels")
		labelsSchema := requireMap(t, labels["schema"], format+" labels schema")
		require.Equal(t, map[string]any{"true": "on", "false": "off"}, labelsSchema["default"], format)
		groups := parameterByName(t, list, "groups")
		groupsSchema := requireMap(t, groups["schema"], format+" groups schema")
		groupSchema := requireMap(t, groupsSchema["additionalProperties"], format+" group schema")
		groupsExample := requireMap(t, groups["example"], format+" groups example")
		nested := make([]any, 0, len(groupsExample)+1)
		nested = append(nested, groupSchema["example"])
		for _, group := range groupsExample {
			nested = append(nested, group)
		}
		for _, group := range nested {
			members := requireMap(t, group, format+" group example")
			require.NotEmpty(t, members, format)
			for key := range members {
				require.Contains(t, []string{"true", "false"}, key, format)
			}
		}

		media := responseMedia(t, spec, "/flags")
		require.Equal(t, map[string]any{"1": "one", "20": "twenty"}, media["example"], format)

		update := requireOperation(t, spec, "/flags", "put")
		body := requireMap(t, update["requestBody"], format+" update request body")
		content := requireMap(t, body["content"], format+" update request content")
		bodyMedia := requireMap(t, content["application/json"], format+" update JSON media")
		example := requireMap(t, bodyMedia["example"], format+" update example")
		rules := requireMap(t, example["rules"], format+" update rules example")
		rule := requireMap(t, rules["true"], format+" update rule example")
		require.Len(t, rule, 1, format)
		require.EqualValues(t, 3, rule["maxCount"], format)
	}
}
