package generator

import (
	"bytes"
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestResponsePublicIdentityPreservesReferenceAnnotations(t *testing.T) {
	root := codegen.RunDSL(t, testdata.UnionResponseBodyDSL)
	roots := []eval.Root{root}
	_, err := Service("example.com/valueprobe/gen", roots)
	require.NoError(t, err)
	_, err = Transport("example.com/valueprobe/gen", roots)
	require.NoError(t, err)
	files, err := OpenAPI("example.com/valueprobe/gen", roots)
	require.NoError(t, err)
	var document map[string]any
	for _, file := range files {
		if !strings.HasSuffix(file.Path, "openapi.json") {
			continue
		}
		var data bytes.Buffer
		for _, section := range file.AllSections() {
			require.NoError(t, section.Write(&data))
		}
		require.NoError(t, json.Unmarshal(data.Bytes(), &document))
	}
	require.NotNil(t, document)
	components := document["components"].(map[string]any)
	componentSchemas := components["schemas"].(map[string]any)
	require.Contains(t, componentSchemas, "NamedChoice")
	require.Contains(t, componentSchemas, "NamedChoiceLeafEnvelope")
	require.Contains(t, componentSchemas, "NamedChoiceOtherEnvelope")
	require.NotContains(t, components, "responses", "responses with different retained examples must remain inline")

	paths := document["paths"].(map[string]any)
	var commonSchema map[string]any
	examples := make(map[string]struct{})
	for _, path := range []string{"/required_nullable", "/optional_nullable", "/required_named_nullable", "/optional_named_nullable"} {
		response := paths[path].(map[string]any)["get"].(map[string]any)["responses"].(map[string]any)["400"].(map[string]any)
		require.NotContains(t, response, "$ref")
		media := response["content"].(map[string]any)["application/json"].(map[string]any)
		schema := media["schema"].(map[string]any)
		require.Equal(t, media["example"], schema["example"])

		example := schema["example"].(map[string]any)
		tag := example["type"].(string)
		value := example["value"].(map[string]any)
		switch tag {
		case "Leaf":
			require.IsType(t, "", value["name"])
		case "Other":
			require.IsType(t, float64(0), value["count"])
		default:
			t.Fatalf("unexpected union example tag %q", tag)
		}
		encoded, err := json.Marshal(example, json.Deterministic(true))
		require.NoError(t, err)
		examples[string(encoded)] = struct{}{}

		withoutExample := make(map[string]any, len(schema)-1)
		for key, value := range schema {
			if key != "example" {
				withoutExample[key] = value
			}
		}
		if commonSchema == nil {
			commonSchema = withoutExample
		} else {
			require.Equal(t, commonSchema, withoutExample)
		}
	}
	require.Len(t, examples, 4, "each inline response retains its occurrence-specific example")

	anyOf := commonSchema["anyOf"].([]any)
	require.Len(t, anyOf, 2)
	require.Equal(t, map[string]any{"type": "null"}, anyOf[1])
	union := anyOf[0].(map[string]any)
	require.Equal(t, []any{
		map[string]any{"$ref": "#/components/schemas/NamedChoiceLeafEnvelope"},
		map[string]any{"$ref": "#/components/schemas/NamedChoiceOtherEnvelope"},
	}, union["oneOf"])
	require.Equal(t, map[string]any{
		"propertyName": "type",
		"mapping": map[string]any{
			"Leaf":  "#/components/schemas/NamedChoiceLeafEnvelope",
			"Other": "#/components/schemas/NamedChoiceOtherEnvelope",
		},
	}, union["discriminator"])
}
