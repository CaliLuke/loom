package openapiimport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportedParameterShapeAdmission(t *testing.T) {
	for _, loc := range []string{"query", "header", "path", "cookie"} {
		for _, tc := range []struct {
			name, schema string
			allowed      bool
		}{
			{"object", `{"type":"object","properties":{"a":{"type":"string"}}}`, false},
			{"nested_array", `{"type":"array","items":{"type":"array","items":{"type":"string"}}}`, false},
			{"primitive", `{"type":"string"}`, true},
			{"primitive_array", `{"type":"array","items":{"type":"string"}}`, loc != "cookie"},
		} {
			t.Run(loc+"/"+tc.name, func(t *testing.T) {
				route := "/value"
				if loc == "path" {
					route = "/value/{q}"
				}
				source := fmt.Appendf(nil, `{"openapi":"3.1.1","info":{"title":"Shape","version":"1"},"paths":{%q:{"get":{"operationId":"value","parameters":[{"name":"q","in":%q,"required":true,"schema":%s}],"responses":{"204":{"description":"ok"}}}}}}`, route, loc, tc.schema)
				document, diags, err := Analyze(source)
				require.NoError(t, err)
				fatal, _ := diags.Classify(true)
				if tc.allowed {
					require.Empty(t, fatal)
					_, err = Render(document, Options{PackageName: "design"})
					require.NoError(t, err)
					return
				}
				assertBindingDiagnostic(t, fatal, "/parameters/0")
				_, err = Render(document, Options{PackageName: "design"})
				if err == nil {
					t.Error("direct Render must reject unsupported binding")
				}
			})
		}
	}
}

func TestImportedParameterShapeRepresentations(t *testing.T) {
	for _, tc := range []struct {
		name, location, schema string
		json                   bool
		allowed                bool
	}{
		{"closed query record", "query", `{"type":"object","additionalProperties":false}`, true, false},
		{"omitted open query map", "query", `{"type":"object"}`, true, true},
		{"explicit open query map", "query", `{"type":"object","additionalProperties":true}`, true, true},
		{"typed query map", "query", `{"type":"object","additionalProperties":{"type":"array","items":{"type":"string"}}}`, true, true},
		{"header map", "header", `{"type":"object","additionalProperties":{"type":"string"}}`, false, false},
		{"path map", "path", `{"type":"object","additionalProperties":{"type":"string"}}`, false, false},
		{"cookie map", "cookie", `{"type":"object","additionalProperties":{"type":"string"}}`, false, false},
		{"header Any", "header", `{}`, false, true},
		{"cookie Any", "cookie", `{}`, false, true},
		{"path Any", "path", `{}`, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			route := "/value"
			if tc.location == "path" {
				route += "/{q}"
			}
			value := `"schema":` + tc.schema
			if tc.json {
				value = `"content":{"application/json":{"schema":` + tc.schema + `}}`
			}
			source := fmt.Appendf(nil, `{"openapi":"3.1.1","info":{"title":"Shape","version":"1"},"paths":{%q:{"get":{"operationId":"value","parameters":[{"name":"q","in":%q,"required":true,%s}],"responses":{"204":{"description":"ok"}}}}}}`, route, tc.location, value)
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			fatal, _ := diagnostics.Classify(true)
			_, renderErr := Render(document, Options{})
			if tc.allowed {
				require.Empty(t, fatal)
				require.NoError(t, renderErr)
			} else {
				assertBindingDiagnostic(t, fatal, "/parameters/0")
				if renderErr == nil {
					t.Error("direct Render must reject unsupported representation")
				}
			}
		})
	}
}

func TestImportedParameterShapeReferencesAndSelection(t *testing.T) {
	for _, tc := range []struct {
		name, item string
		allowed    bool
	}{
		{"scalar item alias", `{type: string}`, true},
		{"nested item alias", `{type: array, items: {type: string}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(fmt.Sprintf(`openapi: 3.1.1
info: {title: References, version: '1'}
paths:
  /selected:
    get:
      operationId: selected
      parameters: [{$ref: '#/components/parameters/Q'}]
      responses: {'204': {description: ok}}
  /ordinary:
    get:
      operationId: ordinary
      responses: {'204': {description: ok}}
components:
  parameters:
    Q: {name: q, in: header, schema: {$ref: '#/components/schemas/Values'}}
  schemas:
    Values: {type: array, items: {$ref: '#/components/schemas/ItemValue'}}
    ItemValue: %s
`, tc.item))
			document, diagnostics, _, err := AnalyzeSelected(source, Selection{Paths: []string{"/selected"}})
			require.NoError(t, err)
			fatal, _ := diagnostics.Classify(true)
			_, renderErr := Render(document, Options{})
			if tc.allowed {
				require.Empty(t, fatal)
				require.NoError(t, renderErr)
			} else {
				assertBindingDiagnostic(t, fatal, "/parameters/0")
				if renderErr == nil {
					t.Error("direct Render must reject referenced unsupported item")
				}
			}
			ordinary, diagnostics, _, err := AnalyzeSelected(source, Selection{Paths: []string{"/ordinary"}})
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			_, err = Render(ordinary, Options{})
			require.NoError(t, err)
		})
	}
}

func TestImportedResponseHeaderShapeAdmission(t *testing.T) {
	for _, tc := range []struct {
		name, schema string
		allowed      bool
	}{
		{"scalar", `{type: string}`, true},
		{"scalar list", `{type: array, items: {type: string}}`, true},
		{"record", `{type: object, properties: {name: {type: string}}}`, false},
		{"nested array", `{type: array, items: {type: array, items: {type: string}}}`, false},
		{"map", `{type: object, additionalProperties: {type: string}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(fmt.Sprintf(`openapi: 3.1.1
info: {title: Headers, version: '1'}
paths:
  /value:
    get:
      operationId: value
      responses:
        '200':
          description: ok
          headers:
            X-Value: {schema: %s}
`, tc.schema))
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			fatal, _ := diagnostics.Classify(true)
			_, renderErr := Render(document, Options{})
			if tc.allowed {
				require.Empty(t, fatal)
				require.NoError(t, renderErr)
			} else {
				assertBindingDiagnostic(t, fatal, "/headers/X-Value")
				if renderErr == nil {
					t.Error("direct Render must reject unsupported response header")
				}
			}
		})
	}
}

func TestImportedParameterShapeDirectDocument(t *testing.T) {
	source := []byte(`{"openapi":"3.1.1","info":{"title":"Direct","version":"1"},"paths":{"/value":{"get":{"operationId":"value","parameters":[{"name":"q","in":"header","schema":{"type":"string"}}],"responses":{"204":{"description":"ok"}}}}}}`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	document.Operations[0].Parameters[0].Schema = &Schema{Type: "array", Items: &Schema{Type: "array", Items: &Schema{Type: "string"}}}
	_, err = Render(document, Options{})
	require.Error(t, err, "public normalized documents must pass the same location admission")
	require.Contains(t, err.Error(), "/parameters/0")
}

func assertBindingDiagnostic(t *testing.T, diagnostics Diagnostics, occurrence string) {
	t.Helper()
	var matches int
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Path, occurrence) {
			matches++
			require.Equal(t, "parameter-shape", diagnostic.Code)
		}
	}
	require.Equal(t, 1, matches, "one source-qualified binding diagnostic at %s, got %v", occurrence, diagnostics)
}
