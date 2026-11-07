package openapiimport

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

func requireOneOfSpec(t *testing.T, source []byte, moduleDir, name string) {
	t.Helper()
	generated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	parsed, err := libopenapi.NewDocument(generated)
	require.NoError(t, err)
	_, err = parsed.BuildV3Model()
	require.NoError(t, err)
	var original, output map[string]any
	require.NoError(t, json.Unmarshal(source, &original))
	require.NoError(t, json.Unmarshal(generated, &output))
	if name == "40-form-captured-type" {
		operation := output["paths"].(map[string]any)["/value"].(map[string]any)["post"].(map[string]any)
		media := operation["requestBody"].(map[string]any)["content"].(map[string]any)["application/x-www-form-urlencoded"].(map[string]any)
		encodings, _ := media["encoding"].(map[string]any)
		encoding, _ := encodings["value"].(map[string]any)
		if encoding["contentType"] != nil {
			require.Equal(t, "application/json", encoding["contentType"])
		} else {
			resolve := func(schema map[string]any) map[string]any {
				for schema["$ref"] != nil {
					name := strings.TrimPrefix(schema["$ref"].(string), "#/components/schemas/")
					schema = output["components"].(map[string]any)["schemas"].(map[string]any)[name].(map[string]any)
				}
				return schema
			}
			body := resolve(media["schema"].(map[string]any))
			value := resolve(body["properties"].(map[string]any)["value"].(map[string]any))
			require.Equal(t, "object", value["type"], "export must preserve the source whole-object JSON default")
		}
		require.NotContains(t, encoding, "style")
		require.NotContains(t, encoding, "explode")
		require.NotContains(t, encoding, "allowReserved")
	}
	// Draft 7 and 2020-12 share these type, enum, nesting and exact-one cases.
	// Closure is covered structurally here and behaviorally by the union runtime.
	if name == "41-closure-count-first" {
		schema := output["paths"].(map[string]any)["/value"].(map[string]any)["post"].(map[string]any)["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
		for schema["$ref"] != nil {
			component := strings.TrimPrefix(schema["$ref"].(string), "#/components/schemas/")
			schema = output["components"].(map[string]any)["schemas"].(map[string]any)[component].(map[string]any)
		}
		require.Contains(t, schema, "oneOf")
		require.Equal(t, false, schema["unevaluatedProperties"])
		return
	}
	if values := oneOfSemanticCases(name); len(values) > 0 {
		requireOneOfSemanticValues(t, original, name, values)
		requireOneOfSemanticValues(t, output, name, values)
		return
	}
	values := []any{nil, "a", "b", "other", 1, 1.5, true, []any{}, map[string]any{}, map[string]any{"x": "a"}, map[string]any{"x": 1}, map[string]any{"y": "a"}, map[string]any{"z": "a"}, map[string]any{"x": "a", "y": "b"}}
	var expected []bool
	for _, document := range []map[string]any{original, output} {
		schema := oneOfSpecTarget(t, document, name)
		compiled, err := gojsonschema.NewSchema(gojsonschema.NewGoLoader(map[string]any{"allOf": []any{schema}, "components": document["components"]}))
		require.NoError(t, err)
		actual := make([]bool, len(values))
		for index, value := range values {
			result, err := compiled.Validate(gojsonschema.NewGoLoader(value))
			require.NoError(t, err)
			actual[index] = result.Valid()
		}
		if expected == nil {
			expected = actual
		} else {
			require.Equal(t, expected, actual, "source and regenerated JSON membership disagree for %#v", values)
		}
	}
}

func resolveOneOfSpecReference(t *testing.T, document, schema map[string]any) map[string]any {
	t.Helper()
	seen := make(map[string]bool)
	for schema["$ref"] != nil {
		reference := schema["$ref"].(string)
		require.False(t, seen[reference], "cyclic top-level schema reference")
		seen[reference] = true
		schema = oneOfSpecReferenceTarget(t, document, reference)
	}
	return schema
}

func oneOfSpecReferenceTarget(t *testing.T, document map[string]any, reference string) map[string]any {
	t.Helper()
	require.True(t, strings.HasPrefix(reference, "#/components/schemas/"))
	name := strings.TrimPrefix(reference, "#/components/schemas/")
	return document["components"].(map[string]any)["schemas"].(map[string]any)[name].(map[string]any)
}
