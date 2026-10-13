package openapiimport

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaAnnotationsAcrossConstructors(t *testing.T) {
	for _, shape := range []struct {
		name   string
		schema *Schema
	}{
		{"string", &Schema{Type: "string"}},
		{"integer", &Schema{Type: "integer"}},
		{"enum", &Schema{Type: "string", Enum: []any{"one", "two"}}},
		{"array", &Schema{Type: "array", Items: &Schema{Type: "string"}}},
		{"map", &Schema{Type: "object", AdditionalProperties: &AdditionalProperties{Schema: &Schema{Type: "string"}}}},
		{"object", &Schema{Type: "object", Properties: []NamedProperty{{Name: "value", Schema: &Schema{Type: "string"}}}}},
	} {
		for _, role := range []string{"attribute", "named", "body", "array-item", "map-value"} {
			if shape.name == "object" && (role == "array-item" || role == "map-value") {
				continue // Structured collection elements are promoted before rendering.
			}
			t.Run(shape.name+"/"+role, func(t *testing.T) {
				schema := *shape.schema
				schema.Description = "Authored schema description."
				schema.Title = "Authored schema title"
				r := renderer{}
				var err error
				switch role {
				case "attribute":
					err = r.attribute("value", &schema, nil, "#/value")
				case "named":
					err = r.namedSchema(NamedSchema{Name: "Value", GoName: "Value", Schema: &schema})
				case "body":
					err = r.responseSchemaBody("OpenAPIBody", &schema, "#/value")
				case "array-item":
					err = r.attribute("values", &Schema{Type: "array", Items: &schema}, nil, "#/values")
				case "map-value":
					err = r.attribute("values", &Schema{Type: "object", AdditionalProperties: &AdditionalProperties{Schema: &schema}}, nil, "#/values")
				}
				require.NoError(t, err)
				output := r.builder.String()
				require.Equal(t, 1, strings.Count(output, `Title("Authored schema title")`), output)
				require.Equal(t, 1, strings.Count(output, `Description("Authored schema description.")`), output)
			})
		}
	}
}

func TestSchemaAnnotationsSupportedFixtureRoundTrip(t *testing.T) {
	var source map[string]any
	require.NoError(t, json.Unmarshal(readFixture(t, "supported.json"), &source))
	schemas := source["components"].(map[string]any)["schemas"].(map[string]any)
	pet := schemas["Pet"].(map[string]any)
	properties := pet["properties"].(map[string]any)
	annotations := map[string]map[string]any{
		"record":               pet,
		"scalar":               properties["name"].(map[string]any),
		"integer":              properties["stock"].(map[string]any),
		"array":                properties["labels"].(map[string]any),
		"item":                 properties["labels"].(map[string]any)["items"].(map[string]any),
		"map":                  properties["metadata"].(map[string]any),
		"value":                properties["metadata"].(map[string]any)["additionalProperties"].(map[string]any),
		"alias":                schemas["Status"].(map[string]any),
		"reference occurrence": properties["status"].(map[string]any),
		"constrained":          schemas["NewPet"].(map[string]any)["properties"].(map[string]any)["name"].(map[string]any),
	}
	for name, schema := range annotations {
		schema["description"] = name + " description"
		schema["title"] = name + " title"
	}
	operation := source["paths"].(map[string]any)["/pets/{pet_id}"].(map[string]any)["get"].(map[string]any)
	parameters := operation["parameters"].([]any)
	for index, raw := range parameters {
		parameter := raw.(map[string]any)
		parameter["schema"].(map[string]any)["description"] = "parameter schema description"
		if index == 0 {
			parameter["description"] = "outer parameter description"
			parameter["schema"] = map[string]any{
				"allOf":       []any{map[string]any{"$ref": "#/components/schemas/Status"}},
				"description": "parameter schema description",
			}
		}
	}
	header := operation["responses"].(map[string]any)["200"].(map[string]any)["headers"].(map[string]any)["X-Trace-ID"].(map[string]any)
	header["schema"].(map[string]any)["description"] = "header schema description"
	encoded, err := json.Marshal(source, json.Deterministic(true))
	require.NoError(t, err)
	document, diagnostics, err := Analyze(encoded)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	contract := readGeneratedOpenAPIContract(t, requireRenderedDesignGenerates(t, rendered))
	gotSchemas := contract["components"].(map[string]any)["schemas"].(map[string]any)
	gotPet := gotSchemas["Pet"].(map[string]any)
	gotProperties := gotPet["properties"].(map[string]any)
	got := map[string]map[string]any{
		"record":               gotPet,
		"scalar":               gotProperties["name"].(map[string]any),
		"integer":              gotProperties["stock"].(map[string]any),
		"array":                gotProperties["labels"].(map[string]any),
		"item":                 gotProperties["labels"].(map[string]any)["items"].(map[string]any),
		"map":                  gotProperties["metadata"].(map[string]any),
		"value":                gotProperties["metadata"].(map[string]any)["additionalProperties"].(map[string]any),
		"alias":                gotSchemas["Status"].(map[string]any),
		"reference occurrence": gotProperties["status"].(map[string]any),
		"constrained":          gotSchemas["NewPet"].(map[string]any)["properties"].(map[string]any)["name"].(map[string]any),
	}
	for name, schema := range got {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, name+" title", schema["title"])
			require.Equal(t, name+" description", schema["description"])
		})
	}
	gotOperation := operationFromImportedSpec(t, contract, "/pets/{pet_id}", "get")
	for _, raw := range gotOperation["parameters"].([]any) {
		parameter := raw.(map[string]any)
		name, _ := parameter["name"].(string)
		if name != "limit" && name != "X-Request-ID" {
			continue
		}
		t.Run("parameter/"+name, func(t *testing.T) {
			outer, _ := parameter["description"].(string)
			if name == "limit" {
				require.Equal(t, "outer parameter description", outer)
			} else {
				require.Empty(t, outer)
			}
			require.Equal(t, "parameter schema description", parameter["schema"].(map[string]any)["description"])
		})
	}
	t.Run("header", func(t *testing.T) {
		header := gotOperation["responses"].(map[string]any)["200"].(map[string]any)["headers"].(map[string]any)["X-Trace-ID"].(map[string]any)
		require.Equal(t, "Request trace identifier.", header["description"])
		require.Equal(t, "header schema description", header["schema"].(map[string]any)["description"])
	})
}
