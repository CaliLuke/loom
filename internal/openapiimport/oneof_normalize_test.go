package openapiimport

import (
	"testing"

	"github.com/CaliLuke/loom/internal/httpencoding"

	"github.com/stretchr/testify/require"
)

func TestOneOfNormalizedFacts(t *testing.T) {
	for _, tc := range []struct {
		name, schema     string
		branches         int
		nullable, closed bool
	}{
		{"one null", `{"oneOf":[{"type":"string"},{"type":"integer"},{"type":"null"}]}`, 2, true, false},
		{"duplicate null", `{"oneOf":[{"type":"string"},{"type":"integer"},{"type":"null"},{"type":"null"}]}`, 2, false, false},
		{"local null stripped", `{"oneOf":[{"type":["string","null"]},{"type":"integer"},{"type":"null"}]}`, 2, false, false},
		{"declared Any null", `{"oneOf":[{}, {"type":"string"}]}`, 2, false, false},
		{"closure independent", `{"oneOf":[{"type":"string"},{"type":"integer"}],"unevaluatedProperties":false}`, 2, false, true},
		{"singleton", `{"oneOf":[{"type":"string","enum":["x"],"description":"inner"}],"title":"outer"}`, 0, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze(schemaAdmissionSource("3.1.1", tc.schema, "component"))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			schema := document.Components.Schemas[0].Schema
			require.Len(t, schema.OneOf, tc.branches)
			require.Equal(t, tc.nullable, schema.Nullable)
			require.Equal(t, tc.closed, schema.UnionClosed)
			if tc.name == "local null stripped" {
				require.False(t, schema.OneOf[0].Nullable)
				require.Empty(t, schema.OneOf[0].TypeUnion)
			}
			if tc.name == "singleton" {
				require.Equal(t, "inner", schema.Description)
				require.Equal(t, "outer", schema.Title)
				require.Equal(t, []any{"x"}, schema.Enum)
			}
		})
	}
}

func TestOneOfSiblingAdmission(t *testing.T) {
	for _, tc := range []struct {
		name, schema string
		accepted     bool
	}{
		{"integer subset number", `{"oneOf":[{"type":"integer"},{"type":"number"}],"type":"number"}`, true},
		{"null not subset string", `{"oneOf":[{"type":"string"},{"type":"null"}],"type":"string"}`, false},
		{"required retained by every branch", `{"oneOf":[{"type":"object","required":["x"],"properties":{"x":{"type":"string"}}},{"type":"object","required":["x"],"properties":{"x":{"type":"integer"}}}],"required":["x"]}`, true},
		{"required restricts branch", `{"oneOf":[{"type":"object","properties":{"x":{"type":"string"}}},{"type":"object","required":["x"],"properties":{"x":{"type":"integer"}}}],"required":["x"]}`, false},
		{"identical property assertion", `{"oneOf":[{"type":"object","properties":{"x":{"type":"string","minLength":1}}},{"type":"object","additionalProperties":false}],"properties":{"x":{"type":"string"}}}`, true},
		{"property annotation cannot disappear", `{"oneOf":[{"type":"object","properties":{"x":{"type":"string","title":"x"}}},{"type":"object","additionalProperties":false}],"properties":{"x":{"type":"string","title":"x"}}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, diagnostics, err := Analyze(schemaAdmissionSource("3.1.1", tc.schema, "component"))
			require.NoError(t, err)
			if tc.accepted {
				require.Empty(t, diagnostics)
			} else {
				requireDiagnosticCode(t, diagnostics, "schema-composition")
			}
		})
	}
}

func TestOneOfSiblingForwardUnionReference(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Forward, version: "1"}
paths: {}
components:
  schemas:
    A: {type: object, oneOf: [{$ref: '#/components/schemas/Z'}, {type: object, properties: {c: {type: string}}}]}
    Z: {oneOf: [{type: object, properties: {a: {type: string}}}, {type: object, properties: {b: {type: string}}}]}
`)
	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
}

func TestOneOfNullableCodecProjection(t *testing.T) {
	planner := documentPlanner{schemas: map[string]NamedSchema{
		"Base":     {Name: "Base", Schema: &Schema{OneOf: []*Schema{{Type: "string"}, {Type: "integer"}}}},
		"Nullable": {Name: "Nullable", Schema: &Schema{Ref: "#/components/schemas/Base", Nullable: true}},
	}}
	shape := planner.urlShape(&Schema{Ref: "#/components/schemas/Nullable"})
	require.Equal(t, httpencoding.Union, shape.Shape)
	require.True(t, shape.Nullable)
	_, _, err := httpencoding.Resolve(httpencoding.NamedQuery, "value", shape, httpencoding.Serialization{})
	require.ErrorContains(t, err, "nullable")
	_, _, err = httpencoding.Resolve(httpencoding.FormField, "value", shape, httpencoding.Serialization{MediaType: "application/json"})
	require.NoError(t, err)
}

func TestOneOfDeclaredAnyNull(t *testing.T) {
	for _, declaration := range []string{"{}", "true"} {
		t.Run(declaration, func(t *testing.T) {
			source := []byte(`openapi: 3.1.1
info: {title: Declared, version: "1"}
paths: {}
components:
  schemas:
    Anything: ` + declaration + `
    Union: {oneOf: [{$ref: '#/components/schemas/Anything'}, {type: "null"}]}
`)
			_, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			requireDiagnosticCode(t, diagnostics, "schema-type")
		})
	}
}

func TestOneOfCapturedCodecThroughRootReference(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Referenced form, version: "1"}
paths:
  /value:
    post:
      requestBody:
        required: true
        content:
          application/x-www-form-urlencoded:
            schema: {$ref: '#/components/schemas/Choice'}
      responses: {"204": {description: done}}
components:
  schemas:
    Choice:
      oneOf:
        - type: object
          properties:
            value:
              type: array
              oneOf:
                - {type: array, items: {type: string}}
                - {type: array, items: {type: integer}}
        - {type: object, properties: {other: {type: string}}}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), `Meta("http:encoding:sourceType", "array")`)
}

func TestOneOfPrimitiveFormatPreserved(t *testing.T) {
	document, diagnostics, err := Analyze(schemaAdmissionSource("3.1.1", `{"oneOf":[{"type":"string","format":"uuid"},{"type":"integer"}]}`, "component"))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), "Format(FormatUUID)")
}
