package openapiimport

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaExamplesSupportedFixtureRoundTrip(t *testing.T) {
	var source map[string]any
	require.NoError(t, json.Unmarshal(readFixture(t, "supported.json"), &source))
	schemas := source["components"].(map[string]any)["schemas"].(map[string]any)
	pet := schemas["Pet"].(map[string]any)
	name := pet["properties"].(map[string]any)["name"].(map[string]any)
	name["type"] = []any{"string", "null"}
	want := []any{nil, "red", "blue", "blue"}
	name["examples"] = want
	delete(name, "example")
	operation := operationFromImportedSpec(t, source, "/pets/{pet_id}", "get")
	parameter := operation["parameters"].([]any)[0].(map[string]any)
	parameter["schema"].(map[string]any)["examples"] = []any{1, 2, 1}
	source["paths"].(map[string]any)["/colors"] = map[string]any{
		"get": map[string]any{
			"operationId": "colors.get",
			"responses": map[string]any{
				"200": map[string]any{
					"description": "Colors.",
					"content": map[string]any{
						"application/json": map[string]any{
							"schema":  map[string]any{"type": "string", "examples": []any{"red", "blue", "blue"}},
							"example": "green",
						},
					},
				},
			},
		},
	}
	encoded, err := json.Marshal(source, json.Deterministic(true))
	require.NoError(t, err)
	document, diagnostics, err := Analyze(encoded)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	directory := requireRenderedDesignGenerates(t, rendered)
	contract := readGeneratedOpenAPIContract(t, directory)
	gotPet := contract["components"].(map[string]any)["schemas"].(map[string]any)["Pet"].(map[string]any)
	gotName := gotPet["properties"].(map[string]any)["name"].(map[string]any)
	require.Equal(t, want, gotName["examples"])
	require.NotContains(t, gotName, "example")
	gotOperation := operationFromImportedSpec(t, contract, "/pets/{pet_id}", "get")
	foundParameter := false
	for _, raw := range gotOperation["parameters"].([]any) {
		parameter := raw.(map[string]any)
		if parameter["name"] == "limit" {
			foundParameter = true
			schema := parameter["schema"].(map[string]any)
			require.Equal(t, []any{float64(1), float64(2), float64(1)}, schema["examples"])
		}
	}
	require.True(t, foundParameter)
	requireIndependentRegeneration(t, directory)
	// Generated document identity and servers are separate unsupported import
	// surfaces; the schema examples must survive strict schema analysis.
	delete(contract, "$self")
	delete(contract, "jsonSchemaDialect")
	delete(contract, "servers")
	generated, err := json.Marshal(contract, json.Deterministic(true))
	require.NoError(t, err)
	second, diagnostics, err := Analyze(generated)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	for _, operation := range second.Operations {
		if operation.Path == "/colors" {
			response := operation.Responses[0].Response
			require.Len(t, response.Schema.Examples, 3)
			require.Equal(t, "red", response.Schema.Examples[0].Value)
			require.Equal(t, "blue", response.Schema.Examples[1].Value)
			require.Equal(t, "blue", response.Schema.Examples[2].Value)
			require.Len(t, response.Examples, 1)
			require.Equal(t, "green", response.Examples[0].Value)
			return
		}
	}
	t.Error("missing colors operation after second import")
}

func TestRenderPreservesSchemaAndMediaExamples(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Examples, version: "1"}
paths:
  /pets:
    post:
      operationId: createPet
      requestBody:
        required: true
        content:
          application/json:
            example: {name: Fido}
            schema: {$ref: '#/components/schemas/Pet'}
      responses:
        "200":
          description: created
          content:
            application/json:
              examples:
                created:
                  summary: Created pet
                  description: A newly-created pet.
                  value: {name: Fido}
              schema: {$ref: '#/components/schemas/Pet'}
components:
  schemas:
    Pet:
      type: object
      required: [name]
      properties:
        name: {type: string, example: Fido}
        kind: {type: string, examples: [dog, cat]}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, rendered, 1)

	design := string(rendered)
	for _, expected := range []string{
		`Value("Fido")`,
		`Example("example-1", func() {`,
		`Value("dog")`,
		`Example("example-2", func() {`,
		`Value("cat")`,
		`Meta("openapi:example:scope", "schema")`,
		`Meta("openapi:example:scope", "media")`,
		`Example("created", func() {`,
		`Meta("openapi:example:summary", "Created pet")`,
		`Description("A newly-created pet.")`,
		`Value(Val{"name": "Fido"})`,
	} {
		require.Contains(t, design, expected)
	}
}

func TestRenderPreservesNullableSchemas(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Nullable, version: "1"}
paths:
  /records:
    post:
      operationId: createRecord
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Record'}
      responses:
        "200":
          description: record
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Record'}
components:
  schemas:
    Record:
      type: object
      required: [required_note, required_child]
      examples:
        - required_note: null
          required_child: null
          nullable_items: [value, null]
          nullable_values: {value: null}
          nullable_objects: [{name: loom}, null]
          nullable_object_values: {value: null}
      properties:
        required_note:
          type: [string, "null"]
          examples: [null]
        optional_count: {type: [integer, "null"]}
        required_child:
          type: [object, "null"]
          properties:
            name: {type: string}
        optional_tags:
          type: [array, "null"]
          items: {type: string}
        nullable_items:
          type: array
          items: {type: [string, "null"]}
        nullable_values:
          type: object
          additionalProperties: {type: [integer, "null"]}
        nullable_objects:
          type: array
          items:
            anyOf:
              - {$ref: '#/components/schemas/Child'}
              - {type: "null"}
        nullable_object_values:
          type: object
          additionalProperties:
            anyOf:
              - {$ref: '#/components/schemas/Child'}
              - {type: "null"}
        optional_label: {$ref: '#/components/schemas/NullableLabel'}
    Child:
      type: object
      properties:
        name: {type: string}
    NullableLabel:
      type: [string, "null"]
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, rendered, 1)
	moduleDir := requireRenderedDesignGenerates(t, rendered)
	requireTypeNullAcceptance(t, source, moduleDir, "Record", []struct {
		value any
		valid bool
	}{
		{map[string]any{"required_note": nil, "required_child": nil}, true},
		{map[string]any{"required_note": "note", "required_child": map[string]any{"name": "child"}, "optional_count": nil, "optional_tags": nil, "nullable_items": []any{"value", nil}, "nullable_values": map[string]any{"value": nil}}, true},
		{nil, false},
		{"record", false},
		{map[string]any{"required_note": nil}, false},
		{map[string]any{"required_note": 1, "required_child": nil}, false},
		{map[string]any{"required_note": nil, "required_child": "child"}, false},
		{map[string]any{"required_note": nil, "required_child": nil, "nullable_items": []any{1}}, false},
	})
	requireIndependentRegeneration(t, moduleDir)
	design := string(rendered)
	require.GreaterOrEqual(t, strings.Count(design, "Nullable()"), 5)
	require.NotContains(t, design, `Meta("openapi:nullable"`)
	require.NotContains(t, design, `Meta("struct:field:type", "loom.Nullable[`)
	require.Contains(t, design, "ArrayOf(String, func() {\n\t\tNullable()")
	require.Contains(t, design, "MapOf(String, Int, func() {\n\t\tElem(func() {\n\t\t\tNullable()")
	require.Contains(t, design, "Null())")
	require.Contains(t, design, "Attribute(\"optional_label\", ImportedNullableLabel, func() {\n\t\tNullable()")
	require.Contains(t, design, `Value(Val{"nullable_items": []any{"value", nil}, "nullable_object_values": Val{"value": nil}, "nullable_objects": []any{Val{"name": "loom"}, nil}, "nullable_values": Val{"value": nil}, "required_child": nil, "required_note": nil})`)
	require.Contains(t, design, "var ImportedNullableLabel = Type(\"NullableLabel\", String, func() {\n\tMeta(\"openapi:typename:canonical\", \"true\")\n\tNullable()")
}

func TestExampleLiteralKeepsNestedNullAsGoNil(t *testing.T) {
	literal, err := exampleLiteral(map[string]any{
		"items": []any{"value", nil},
		"value": nil,
	})
	require.NoError(t, err)
	require.Equal(t, `Val{"items": []any{"value", nil}, "value": nil}`, literal)

	literal, err = exampleLiteral(nil)
	require.NoError(t, err)
	require.Equal(t, "Null()", literal)
}
