package openapiimport

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestImportedValueDomainNormalization(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.1.1", "3.2.0"} {
		for _, tc := range []struct {
			name     string
			schema   string
			enum     []any
			nullable bool
		}{
			{"nullable enum with null", `%s,"enum":["a",null]`, []any{"a"}, true},
			{"nullable enum without null", `%s,"enum":["a"]`, []any{"a"}, false},
			{"nullable mixed enum", `%s,"enum":["a",1,true,null]`, []any{"a"}, true},
			{"nonnullable null member", `"type":"string","enum":["a",null]`, []any{"a"}, false},
			{"integer null member", `"type":"integer","enum":[1,null]`, []any{1}, false},
			{"boolean mixed member", `"type":"boolean","enum":[false,"false",null]`, []any{false}, false},
			{"ordinary enum", `"type":"string","enum":["a"]`, []any{"a"}, false},
			{"nullable no enum", `%s`, nil, true},
		} {
			t.Run(version+"/"+tc.name, func(t *testing.T) {
				schema := tc.schema
				if strings.Contains(schema, "%s") {
					schema = fmt.Sprintf(schema, valueDomainNullableType(version, "string"))
				}
				document, diagnostics, err := Analyze(schemaAdmissionSource(version, "{"+schema+"}", "component"))
				require.NoError(t, err)
				require.Empty(t, diagnostics)
				actual := document.Components.Schemas[0].Schema
				require.Equal(t, tc.enum, actual.Enum)
				require.Equal(t, tc.nullable, actual.Nullable)
				design, err := Render(document, Options{PackageName: "design"})
				require.NoError(t, err)
				require.NotContains(t, string(design), "nil)")
				require.Equal(t, tc.nullable, strings.Contains(string(design), "Nullable()"))
			})
		}
	}
}

func TestImportedValueDomainConst(t *testing.T) {
	for _, tc := range []struct {
		schema string
		value  any
	}{
		{`{"type":["string","null"],"const":"a"}`, "a"},
		{`{"type":"integer","const":1}`, 1},
		{`{"type":"boolean","const":false}`, false},
	} {
		t.Run(tc.schema, func(t *testing.T) {
			document, diagnostics, err := Analyze(schemaAdmissionSource("3.1.1", tc.schema, "component"))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			schema := document.Components.Schemas[0].Schema
			require.Equal(t, []any{tc.value}, schema.Enum)
			require.False(t, schema.Nullable)
		})
	}
}

func TestImportedValueDomainNumericTypeIntersection(t *testing.T) {
	for _, tc := range []struct {
		schema string
		enum   []any
	}{
		{`{"type":"integer","enum":[1.0]}`, []any{float64(1)}},
		{`{"type":"integer","enum":[1,1.5,"1",null]}`, []any{1}},
		{`{"type":"integer","enum":[1,1.00000000000000001]}`, []any{1}},
		{`{"type":"number","enum":[1,1.5]}`, []any{1, float64(1.5)}},
	} {
		t.Run(tc.schema, func(t *testing.T) {
			document, diagnostics, err := Analyze(schemaAdmissionSource("3.1.1", tc.schema, "component"))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			// Normalization may choose an integer carrier for integral decimal
			// input. Assert JSON numeric meaning, not parser Go carrier identity.
			want, err := json.Marshal(tc.enum)
			require.NoError(t, err)
			got, err := json.Marshal(document.Components.Schemas[0].Schema.Enum)
			require.NoError(t, err)
			require.JSONEq(t, string(want), string(got))
		})
	}
}

func TestImportedValueDomainYAMLNumericIdentity(t *testing.T) {
	document, diagnostics, err := Analyze([]byte(`openapi: 3.1.1
info: {title: Exact membership, version: "1"}
paths: {}
components:
  schemas:
    Value:
      type: integer
      enum: [0x20000000000001, 0o10, 0b11, 1.0, 1.00000000000000001]
`))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Equal(t, []any{9007199254740993, 8, 3, 1}, document.Components.Schemas[0].Schema.Enum)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), "Enum(9007199254740993, 8, 3, 1)")
}

func TestImportedValueDomainRejectsUnrepresentable(t *testing.T) {
	for _, schema := range []string{
		`{"type":["string","null"],"enum":[null]}`,
		`{"type":"string","enum":[null]}`,
		`{"type":"string","enum":[]}`,
		`{"type":"string","enum":[1,true]}`,
		`{"type":["string","null"],"const":null}`,
		`{"type":"integer","const":"one"}`,
		`{"type":"number","enum":[1.00000000000000001]}`,
		`{"type":"integer","enum":[9223372036854775808]}`,
	} {
		t.Run(schema, func(t *testing.T) {
			document, diagnostics, err := Analyze(schemaAdmissionSource("3.1.1", schema, "component"))
			require.NoError(t, err)
			require.NotEmpty(t, diagnostics)
			fatal, _ := diagnostics.Classify(true)
			require.NotEmpty(t, fatal, "allow-lossy cannot widen a finite domain")
			_, err = Render(document, Options{PackageName: "design"})
			require.Error(t, err, "normalized IR must retain admission failure")
		})
	}
}

func TestImportedValueDomainOccurrences(t *testing.T) {
	for _, position := range []string{"response", "request", "component", "property", "items", "map"} {
		t.Run(position, func(t *testing.T) {
			document, diagnostics, err := Analyze(schemaAdmissionSource("3.1.1", `{"type":["string","null"],"enum":["a"]}`, position))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			require.Contains(t, string(design), `Enum("a")`)
			require.NotContains(t, string(design), "Nullable()")
		})
	}
}

func TestImportedValueDomainSelection(t *testing.T) {
	schema := `{"type":["string","null"],"enum":[null]}`
	source := strings.Replace(string(schemaAdmissionSource("3.1.1", schema, "component")), `"paths":{`, `"paths":{"/good":{"get":{"operationId":"good","responses":{"204":{"description":"ok"}}}},`, 1)
	document, diagnostics, _, err := AnalyzeSelected([]byte(source), Selection{Paths: []string{"/good"}})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	_, err = Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	_, diagnostics, _, err = AnalyzeSelected([]byte(source), Selection{Paths: []string{"/value"}})
	require.NoError(t, err)
	require.Len(t, diagnostics, 1, "one shared owner reports the selected domain once")
	require.Equal(t, "#/components/schemas/Value/enum", diagnostics[0].Path)
	partial, _, err := AnalyzePartial([]byte(source), Selection{}, true)
	require.NoError(t, err)
	require.Len(t, partial.Skipped, 1)
	require.Equal(t, "/value", partial.Skipped[0].Path)
	require.Len(t, partial.Document.Operations, 1)
	require.Equal(t, "/good", partial.Document.Operations[0].Path)
}

func TestImportedValueDomainRetainsCompositionBoundary(t *testing.T) {
	for _, schema := range []string{
		`{"anyOf":[{"type":"string","enum":["a"]},{"type":"null"}]}`,
		`{"anyOf":[{"type":"string","enum":["a"]},{"type":"null"}],"enum":[]}`,
		`{"anyOf":[{"type":"string","enum":["a"]},{"type":"null"}],"const":"b"}`,
	} {
		t.Run(schema, func(t *testing.T) {
			document, diagnostics, err := Analyze(schemaAdmissionSource("3.1.1", schema, "component"))
			require.NoError(t, err)
			if strings.Contains(schema, `}],`) {
				require.NotEmpty(t, diagnostics, "authored outer membership is not an absent sibling")
				_, err = Render(document, Options{PackageName: "design"})
				require.Error(t, err)
				return
			}
			require.Empty(t, diagnostics)
			actual := document.Components.Schemas[0].Schema
			require.True(t, actual.Nullable, "outer null branch remains valid despite inner non-null enum")
			require.Equal(t, []any{"a"}, actual.Enum)
		})
	}
}

func TestImportedValueDomainRoundTrip(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.1.1"} {
		for _, includeNull := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/null=%t", version, includeNull), func(t *testing.T) {
				source := valueDomainSource(version, includeNull)
				document, diagnostics, err := Analyze(source)
				require.NoError(t, err)
				require.Empty(t, diagnostics)
				design, err := Render(document, Options{PackageName: "design"})
				require.NoError(t, err)
				moduleDir := requireRenderedDesignGenerates(t, design)
				generated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
				require.NoError(t, err)
				parsed, err := libopenapi.NewDocument(generated)
				require.NoError(t, err)
				_, parseErr := parsed.BuildV3Model()
				require.NoError(t, parseErr)
				requireIndependentRegeneration(t, moduleDir)
				// Draft 7 does not understand OAS 3.0 nullable. Use the exact
				// type-union equivalent as the existing instance-check oracle.
				oracle := valueDomainSource("3.1.1", includeNull)
				requireTypeNullAcceptance(t, oracle, moduleDir, "Record", []struct {
					value any
					valid bool
				}{
					{map[string]any{}, true},
					{map[string]any{"label": "a", "inline": "a", "items": []any{"a"}, "values": map[string]any{"key": "a"}}, true},
					{map[string]any{"label": nil}, includeNull},
					{map[string]any{"inline": nil}, includeNull},
					{map[string]any{"items": []any{nil}}, includeNull},
					{map[string]any{"values": map[string]any{"key": nil}}, includeNull},
					{map[string]any{"label": "other"}, false},
					{map[string]any{"inline": 1}, false},
				})
			})
		}
	}
}

func TestImportedValueDomainNumericRoundTrip(t *testing.T) {
	source := schemaAdmissionSource("3.1.1", `{"type":"object","properties":{"integer":{"type":"integer","format":"int64","enum":[9223372036854775807,1.0,1.00000000000000001]},"small":{"type":"integer","format":"int32","enum":[2147483647]},"decimal":{"type":"number","format":"float","enum":[0.1]},"double":{"type":"number","enum":[1.5]}}}`, "component")
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	moduleDir := requireRenderedDesignGenerates(t, design)
	generated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	var contract struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					Enum []jsontext.Value `json:"enum"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	require.NoError(t, json.Unmarshal(generated, &contract))
	for name, want := range map[string][]string{
		"integer": {"9223372036854775807", "1"},
		"small":   {"2147483647"},
		"decimal": {"0.1"},
		"double":  {"1.5"},
	} {
		values := contract.Components.Schemas["Value"].Properties[name].Enum
		actual := make([]string, 0, len(values))
		for _, value := range values {
			actual = append(actual, string(value))
		}
		require.Equal(t, want, actual, name)
	}
	requireIndependentRegeneration(t, moduleDir)
}

func valueDomainNullableType(version, typ string) string {
	if strings.HasPrefix(version, "3.0.") {
		return fmt.Sprintf(`"type":%q,"nullable":true`, typ)
	}
	return fmt.Sprintf(`"type":[%q,"null"]`, typ)
}

func valueDomainSource(version string, includeNull bool) []byte {
	members := `"a"`
	if includeNull {
		members += `,null`
	}
	value := fmt.Sprintf(`{%s,"enum":[%s]}`, valueDomainNullableType(version, "string"), members)
	return fmt.Appendf(nil, `{"openapi":%q,"info":{"title":"Value domains","version":"1"},"paths":{"/records":{"get":{"operationId":"read","responses":{"200":{"description":"record","content":{"application/json":{"schema":{"$ref":"#/components/schemas/Record"}}}}}}}},"components":{"schemas":{"Choice":%s,"Record":{"type":"object","properties":{"label":{"$ref":"#/components/schemas/Choice"},"inline":%s,"items":{"type":"array","items":%s},"values":{"type":"object","additionalProperties":%s}}}}}}`, version, value, value, value, value)
}
