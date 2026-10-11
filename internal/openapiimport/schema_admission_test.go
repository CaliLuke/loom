package openapiimport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaAdmissionRejectsLossyNormalization(t *testing.T) {
	tests := []struct {
		name   string
		schema string
	}{
		{name: "false schema", schema: `false`},
		{name: "properties without type", schema: `{"properties":{"name":{"type":"string"}}}`},
		{name: "constrained null branch", schema: `{"anyOf":[{"type":"string"},{"type":"null","enum":["forbidden"]}]}`},
		{name: "annotated null branch", schema: `{"anyOf":[{"type":"string"},{"type":"null","description":"null branch"}]}`},
		{name: "modern nullable true", schema: `{"type":"string","nullable":true}`},
		{name: "modern nullable false", schema: `{"type":"string","nullable":false}`},
	}
	for _, version := range []string{"3.1.1", "3.2.0"} {
		for _, test := range tests {
			for _, position := range []string{"response", "request", "component", "property", "items", "map"} {
				if test.schema == "false" && position == "map" {
					// additionalProperties: false closes an object; it is not a
					// schema occurrence accepting no values.
					continue
				}
				t.Run(version+"/"+test.name+"/"+position, func(t *testing.T) {
					document, diagnostics, err := Analyze(schemaAdmissionSource(version, test.schema, position))
					require.NoError(t, err)
					require.NotEmpty(t, diagnostics, "strict analysis must reject a lossy schema")
					fatal, _ := diagnostics.Classify(true)
					require.NotEmpty(t, fatal, "allow-lossy must not weaken the accepted value domain")
					_, err = Render(document, Options{PackageName: "design"})
					require.Error(t, err, "the normalized document must retain the admission blocker")
				})
			}
		}
	}
}

func TestSchemaAdmissionRejectsModernShapesInOpenAPI30(t *testing.T) {
	for _, schema := range []string{`true`, `false`, `{"type":["string","null"]}`} {
		document, diagnostics, err := Analyze(schemaAdmissionSource("3.0.3", schema, "component"))
		require.NoError(t, err)
		require.NotEmpty(t, diagnostics)
		_, err = Render(document, Options{PackageName: "design"})
		require.Error(t, err)
	}
}

func TestSchemaAdmissionPreservesSupportedControls(t *testing.T) {
	tests := []struct {
		name   string
		schema string
	}{
		{name: "true schema", schema: `true`},
		{name: "empty schema", schema: `{}`},
		{name: "explicit object", schema: `{"type":"object","properties":{"name":{"type":"string"}}}`},
		{name: "bare null branch", schema: `{"anyOf":[{"type":"string"},{"type":"null"}]}`},
		{name: "unconstrained null branch", schema: `{"anyOf":[{},{"type":"null"}]}`},
		{name: "nullable type union", schema: `{"type":["string","null"]}`},
		{name: "closed object", schema: `{"type":"object","additionalProperties":false}`},
		{name: "free form map", schema: `{"type":"object","additionalProperties":true}`},
	}
	for _, version := range []string{"3.1.1", "3.2.0"} {
		for _, test := range tests {
			t.Run(version+"/"+test.name, func(t *testing.T) {
				document, diagnostics, err := Analyze(schemaAdmissionSource(version, test.schema, "component"))
				require.NoError(t, err)
				require.Empty(t, diagnostics)
				_, err = Render(document, Options{PackageName: "design"})
				require.NoError(t, err)
			})
		}
	}
	for _, nullable := range []string{"true", "false"} {
		t.Run("3.0.3/nullable "+nullable, func(t *testing.T) {
			document, diagnostics, err := Analyze(schemaAdmissionSource("3.0.3", `{"type":"string","nullable":`+nullable+`}`, "component"))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			_, err = Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
		})
	}
}

func TestSchemaAdmissionSelectionDropsUnselectedBlockers(t *testing.T) {
	for _, schema := range []string{`false`, `{"properties":{"name":{"type":"string"}}}`, `{"type":"string","nullable":true}`} {
		source := strings.Replace(string(schemaAdmissionSource("3.1.1", schema, "component")), `"paths":{`, `"paths":{"/good":{"get":{"operationId":"good","responses":{"200":{"description":"ok","content":{"application/json":{"schema":{"type":"string"}}}}}}},`, 1)
		document, diagnostics, _, err := AnalyzeSelected([]byte(source), Selection{Paths: []string{"/good"}})
		require.NoError(t, err)
		require.Empty(t, diagnostics)
		require.Empty(t, document.Components.Schemas)
		_, err = Render(document, Options{PackageName: "design"})
		require.NoError(t, err)
		_, diagnostics, _, err = AnalyzeSelected([]byte(source), Selection{Paths: []string{"/value"}})
		require.NoError(t, err)
		require.NotEmpty(t, diagnostics)
		partial, _, err := AnalyzePartial([]byte(source), Selection{}, true)
		require.NoError(t, err)
		require.Len(t, partial.Skipped, 1)
		require.Equal(t, "/value", partial.Skipped[0].Path)
		require.Len(t, partial.Document.Operations, 1)
		require.Equal(t, "/good", partial.Document.Operations[0].Path)
		_, err = Render(partial.Document, Options{PackageName: "design"})
		require.NoError(t, err)
	}
}

func TestSchemaAdmissionBooleanRoundTrip(t *testing.T) {
	for _, version := range []string{"3.1.1", "3.2.0"} {
		t.Run(version, func(t *testing.T) {
			source := strings.Replace(unconstrainedSchemaSource, "3.1.1", version, 1)
			booleanSource := strings.Replace(source, "Anything: {}", "Anything: true", 1)
			booleanSource = strings.Replace(booleanSource, "value: {}", "value: true", 1)
			emptyDocument, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			booleanDocument, diagnostics, err := Analyze([]byte(booleanSource))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			emptyDesign, err := Render(emptyDocument, Options{PackageName: "design"})
			require.NoError(t, err)
			booleanDesign, err := Render(booleanDocument, Options{PackageName: "design"})
			require.NoError(t, err)
			require.Equal(t, emptyDesign, booleanDesign, "true and {} accept the same JSON values")
			requireUnconstrainedRoundTripAndRuntime(t, []byte(booleanSource))
		})
	}
}

func schemaAdmissionSource(version, schema, position string) []byte {
	components := ""
	if position == "property" {
		schema = `{"type":"object","properties":{"value":` + schema + `}}`
	}
	if position == "items" {
		schema = `{"type":"array","items":` + schema + `}`
	}
	if position == "map" {
		schema = `{"type":"object","additionalProperties":` + schema + `}`
	}
	if position == "component" {
		components = `,"components":{"schemas":{"Value":` + schema + `}}`
		schema = `{"$ref":"#/components/schemas/Value"}`
	}
	if position == "request" {
		return fmt.Appendf(nil, `{"openapi":%q,"info":{"title":"Admission","version":"1"},"paths":{"/value":{"post":{"operationId":"value","requestBody":{"required":true,"content":{"application/json":{"schema":%s}}},"responses":{"204":{"description":"ok"}}}}}}`, version, schema)
	}
	return fmt.Appendf(nil, `{"openapi":%q,"info":{"title":"Admission","version":"1"},"paths":{"/value":{"get":{"operationId":"value","responses":{"200":{"description":"value","content":{"application/json":{"schema":%s}}}}}}}%s}`, version, schema, components)
}
