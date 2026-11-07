package openapiimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/httpencoding"
)

func TestEncodingSourceTypeSurvivesNormalization(t *testing.T) {
	for _, tc := range []struct{ name, schema, want string }{
		{"array union", `{"type":"array","oneOf":[{"type":"array","items":{"type":"string"}},{"type":"array","items":{"type":"integer"}}]}`, "array"},
		{"typeless arrays", `{"oneOf":[{"type":"array","items":{"type":"string"}},{"type":"array","items":{"type":"integer"}}]}`, "absent"},
		{"singleton array", `{"oneOf":[{"type":"array","items":{"type":"string"}}]}`, "absent"},
		{"nullable array", `{"type":["array","null"],"items":{"type":"string"}}`, "array"},
		{"nullable composition", `{"anyOf":[{"type":"array","items":{"type":"string"}},{"type":"null"}]}`, "absent"},
		{"object union", `{"type":"object","oneOf":[{"type":"object","properties":{"x":{"type":"string"}}},{"type":"object","properties":{"y":{"type":"integer"}}}]}`, "object"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, _, err := analyzeSelectedDocument(schemaAdmissionSource("3.1.1", tc.schema, "component"), Selection{})
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			require.Equal(t, tc.want, document.Components.Schemas[0].Schema.sourceType)
		})
	}
}

func TestEncodingSourceTypeThroughAllOf(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Source facts, version: "1"}
paths: {}
components:
  schemas:
    A: {allOf: [{$ref: '#/components/schemas/B'}]}
    B: {allOf: [{$ref: '#/components/schemas/C'}]}
    C: {type: array, items: {type: string}}
`)
	document, diagnostics, _, err := analyzeSelectedDocument(source, Selection{})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	for _, named := range document.Components.Schemas {
		require.Equal(t, "array", named.Schema.sourceType, named.Name)
	}
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.NotContains(t, string(design), "http:encoding:sourceType", "plain aliases inherit unchanged source facts")
}

func TestEncodingSourceShapeAndMetadata(t *testing.T) {
	for _, tc := range []struct {
		name, schema, metadata string
		array                  bool
	}{
		{"source array union", `{"type":"array","oneOf":[{"type":"array","items":{"type":"string"}},{"type":"array","items":{"type":"integer"}}]}`, "array", false},
		{"typeless union", `{"oneOf":[{"type":"array","items":{"type":"string"}},{"type":"array","items":{"type":"integer"}}]}`, "", false},
		{"typeless singleton", `{"oneOf":[{"type":"array","items":{"type":"string"}}]}`, "absent", true},
		{"nested array", `{"type":"array","items":{"type":"array","items":{"type":"string"}}}`, "", true},
		{"object array", `{"type":"array","items":{"type":"object","properties":{"x":{"type":"string"}}}}`, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze(schemaAdmissionSource("3.1.1", tc.schema, "component"))
			require.NoError(t, err)
			for _, diagnostic := range diagnostics {
				require.Equal(t, "schema-inline-array-item-promoted", diagnostic.Code)
			}
			planner := documentPlanner{schemas: planDocument(document).schemas}
			schema := planner.schemas["Value"].Schema
			shape := planner.urlShape(schema)
			require.Equal(t, tc.metadata, shape.SourceType)
			if tc.array {
				require.NotNil(t, shape.Items)
			} else {
				require.Equal(t, httpencoding.Union, shape.Shape)
				for _, branch := range shape.Branches {
					require.NotNil(t, branch.Items)
				}
			}
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			if tc.metadata != "" {
				require.Contains(t, string(design), `Meta("http:encoding:sourceType", "`+tc.metadata+`")`)
			} else {
				require.NotContains(t, string(design), `http:encoding:sourceType`)
			}
			require.NotContains(t, string(design), "defaultContentType")
		})
	}
}

func TestEncodingSourceMediaMatrix(t *testing.T) {
	for _, media := range []string{"application/x-www-form-urlencoded", "multipart/form-data"} {
		t.Run(media, func(t *testing.T) {
			for _, tc := range []struct {
				name, schema, encoding string
				accepted               bool
			}{
				{"scalar", `{"type":"string"}`, `{}`, true},
				{"object", `{"type":"object","properties":{"x":{"type":"string"}}}`, `{}`, true},
				{"string items", `{"type":"array","items":{"type":"string"}}`, `{}`, true},
				{"object items", `{"type":"array","items":{"type":"object","properties":{"x":{"type":"string"}}}}`, `{}`, true},
				{"array items", `{"type":"array","items":{"type":"array","items":{"type":"string"}}}`, `{}`, true},
				{"JSON string items", `{"type":"array","items":{"type":"string"}}`, `{"value":{"contentType":"application/json"}}`, true},
				{"source array alternatives", `{"type":"array","oneOf":[{"type":"array","items":{"type":"string"}},{"type":"array","items":{"type":"integer"}}]}`, `{}`, true},
				{"mixed item defaults", `{"type":"array","oneOf":[{"type":"array","items":{"type":"string"}},{"type":"array","items":{"type":"object","properties":{"x":{"type":"string"}}}}]}`, `{}`, true},
				{"whole typeless arrays", `{"oneOf":[{"type":"array","items":{"type":"string"}},{"type":"array","items":{"type":"integer"}}]}`, `{"value":{"contentType":"application/json"}}`, true},
				{"unencoded typeless arrays", `{"oneOf":[{"type":"array","items":{"type":"string"}},{"type":"array","items":{"type":"integer"}}]}`, `{}`, false},
				{"JSON union items", `{"type":"array","items":{"oneOf":[{"type":"string"},{"type":"integer"}]}}`, `{"value":{"contentType":"application/json"}}`, true},
				{"nullable JSON items", `{"type":"array","items":{"type":["string","null"]}}`, `{"value":{"contentType":"application/json"}}`, true},
				{"content suppressed by false", `{"type":"array","items":{"type":"string"}}`, `{"value":{"contentType":"application/json","allowReserved":false}}`, true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					schema := `{"type":"object","properties":{"value":` + tc.schema + `}}`
					source := strings.ReplaceAll(string(rootEncodingSource(schema, tc.encoding)), "application/x-www-form-urlencoded", media)
					document, diagnostics, err := Analyze([]byte(source))
					require.NoError(t, err)
					if !tc.accepted {
						requireDiagnosticCode(t, diagnostics, "media-encoding")
						return
					}
					for _, diagnostic := range diagnostics {
						require.Equal(t, "schema-inline-array-item-promoted", diagnostic.Code)
					}
					design, err := Render(document, Options{PackageName: "design"})
					require.NoError(t, err)
					require.NotContains(t, string(design), "defaultContentType")
				})
			}
		})
	}
}

func TestEncodingSourceTypeSingletonReferenceOverride(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Source facts, version: "1"}
paths: {}
components:
  schemas:
    A: {oneOf: [{$ref: '#/components/schemas/B'}]}
    B: {type: array, items: {type: string}}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	planner := documentPlanner{schemas: planDocument(document).schemas}
	shape := planner.urlShape(planner.schemas["A"].Schema)
	require.Equal(t, "absent", shape.SourceType)
	require.NotNil(t, shape.Items)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), `Meta("http:encoding:sourceType", "absent")`)
}

func TestEncodingMultipartNamespaces(t *testing.T) {
	for _, tc := range []struct {
		name, schema, encoding string
		collision              bool
	}{
		{"exploded object", `{"type":"object","properties":{"filter":{"type":"object","properties":{"x":{"type":"string"}}},"x":{"type":"string"}}}`, `{"filter":{"style":"form","explode":true}}`, true},
		{"delimited object", `{"type":"object","properties":{"filter":{"type":"object","properties":{"x":{"type":"string"}}},"x":{"type":"string"}}}`, `{"filter":{"style":"form","explode":false}}`, false},
		{"JSON object", `{"type":"object","properties":{"filter":{"type":"object","properties":{"x":{"type":"string"}}},"x":{"type":"string"}}}`, `{"filter":{"contentType":"application/json"}}`, false},
		{"union wide ownership", `{"type":"object","properties":{"filter":{"oneOf":[{"type":"object","properties":{"x":{"type":"string"}}},{"type":"object","properties":{"y":{"type":"string"}}}]},"y":{"type":"string"}}}`, `{"filter":{"style":"form","explode":true}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.ReplaceAll(string(rootEncodingSource(tc.schema, tc.encoding)), "application/x-www-form-urlencoded", "multipart/form-data")
			_, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			if tc.collision {
				requireDiagnosticCode(t, diagnostics, "media-encoding")
				require.Contains(t, diagnostics.Error(), "overlapping parameter namespaces")
			} else {
				require.Empty(t, diagnostics)
			}
		})
	}
}

func TestEncodingMultipartBinaryApplication(t *testing.T) {
	for _, tc := range []struct {
		name, field string
		accepted    bool
	}{
		{"raw item", `{"type":"array","items":{"type":"string","format":"binary"}}`, true},
		{"raw union branch items", `{"type":"array","oneOf":[{"type":"array","items":{"type":"string","format":"binary"}},{"type":"array","items":{"type":"string"}}]}`, true},
		{"inner array is JSON", `{"type":"array","items":{"type":"array","items":{"type":"string","format":"binary"}}}`, false},
		{"object item is JSON", `{"type":"array","items":{"type":"object","properties":{"raw":{"type":"string","format":"binary"}}}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schema := `{"type":"object","properties":{"value":` + tc.field + `}}`
			source := strings.ReplaceAll(string(rootEncodingSource(schema, `{}`)), "application/x-www-form-urlencoded", "multipart/form-data")
			_, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			if tc.accepted {
				require.Empty(t, diagnostics)
			} else {
				requireDiagnosticCode(t, diagnostics, "schema-binary-location")
			}
		})
	}
}
