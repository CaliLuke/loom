package openapiv3_test

import (
	"encoding/json/v2"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRenderedCollectionExamples(t *testing.T) {
	for _, version := range []struct{ target, want string }{
		{"3.1", openapiv3.OpenAPICompatibilityVersion},
		{"3.2", openapiv3.OpenAPIVersion},
	} {
		t.Run(version.target, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, testdata.CollectionExamplesDSL, version.target, version.want)
			var document map[string]any
			require.NoError(t, json.Unmarshal(artifacts.JSON, &document))
			arrays, maps := requireCollectionExamples(t, document)
			require.Positive(t, arrays, "no labels example rendered")
			require.Positive(t, maps, "no children example rendered")

			operation := requireOperation(t, document, "/counts", "get")
			responses := requireMap(t, operation["responses"], "responses")
			response := requireMap(t, responses["200"], "response")
			content := requireMap(t, response["content"], "content")
			media := requireMap(t, content["application/json"], "JSON media")
			schema := requireMap(t, media["schema"], "array schema")
			items := requireMap(t, schema["items"], "item schema")
			require.Equal(t, "integer", items["type"])
			require.EqualValues(t, 1, items["minimum"])
			require.EqualValues(t, 4, items["maximum"])
			example, ok := schema["example"].([]any)
			require.True(t, ok, "typed alias array example must be retained")
			require.NotEmpty(t, example)
			require.Equal(t, schema["example"], media["example"])
			for _, raw := range example {
				value := raw.(float64)
				require.Equal(t, math.Trunc(value), value)
				require.GreaterOrEqual(t, value, 1.0)
				require.LessOrEqual(t, value, 4.0)
			}
		})
	}
}

func requireCollectionExamples(t *testing.T, value any) (arrays, maps int) {
	t.Helper()
	switch actual := value.(type) {
	case map[string]any:
		for key, member := range actual {
			if labels, ok := member.([]any); key == "labels" && ok {
				arrays++
				require.NotEmpty(t, labels)
				for _, label := range labels {
					require.Equal(t, "", label)
				}
			}
			if children, ok := member.(map[string]any); key == "children" && ok {
				// Schema properties also use this name; only inspect examples.
				if _, schema := children["type"]; !schema {
					maps++
					require.NotEmpty(t, children)
					for _, child := range children {
						require.IsType(t, map[string]any{}, child)
						require.NotNil(t, child)
					}
				}
			}
			a, m := requireCollectionExamples(t, member)
			arrays += a
			maps += m
		}
	case []any:
		for _, member := range actual {
			a, m := requireCollectionExamples(t, member)
			arrays += a
			maps += m
		}
	}
	return arrays, maps
}
