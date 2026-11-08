package openapiimport

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportStatusSelectorAnalyze(t *testing.T) {
	source, err := os.ReadFile("testdata/status_selectors.yaml")
	require.NoError(t, err)
	doc, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Len(t, doc.Operations, 12)
	for _, op := range doc.Operations {
		if op.OperationID == "fallback" {
			require.Len(t, op.Responses, 1)
			require.Equal(t, "default", op.Responses[0].Status)
			require.Equal(t, "#/components/schemas/Value", op.Responses[0].Response.Schema.Ref)
		}
	}
}

func TestImportStatusSelectorRender(t *testing.T) {
	source, err := os.ReadFile("testdata/status_selectors.yaml")
	require.NoError(t, err)
	doc, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(doc, Options{})
	require.NoError(t, err)
	for _, want := range []string{`Status2XX`, `Status3XX`, `Status4XX`, `StatusFallback`, `StatusField("http_status")`, `NoSuccessResponse()`, `Informational("HintsStatus103"`, `Informational("ProgressStatus1XX"`, `Attribute("status_200"`, `Attribute("status_2xx"`} {
		require.Contains(t, string(rendered), want)
	}
	again, err := Render(doc, Options{})
	require.NoError(t, err)
	require.Equal(t, rendered, again)
	requireRenderedDesignEvaluates(t, rendered, 12, `
 for _, endpoint := range expr.Root.API.HTTP.Services[0].HTTPEndpoints {
  if endpoint.MethodExpr.Name == "ErrorsOnly" && (!endpoint.NoSuccess || len(endpoint.Responses) != 0) { t.Error("error-only endpoint has a success") }
  for _, response := range endpoint.Responses {
   if !response.Status.IsExact() && response.StatusField == "" { t.Error("class lacks status field") }
   if endpoint.MethodExpr.Name == "ExactClass" {
    selected := "status_2xx"
    other := "status_200"
    if response.Status.IsExact() { selected, other = other, selected }
    fields := response.RequiredFields("")
    found := false
    for _, field := range fields {
     if field == other { t.Error("another branch became required") }
     if field == selected { found = true }
    }
    if !found { t.Error("selected branch is not required") }
   }
  }
 }
 `)
}

func TestImportStatusSelectorRejections(t *testing.T) {
	cases := []struct{ name, responses, message string }{
		{"upgrade", `"101": {description: upgrade}`, "101 describes a protocol upgrade OpenAPI cannot define"},
		{"lowercase", `"2xx": {description: lower}`, "HTTP code, uppercase 1XX-5XX, or default"},
		{"malformed", `"20X": {description: malformed}`, "HTTP code, uppercase 1XX-5XX, or default"},
		{"range", `"6XX": {description: outside}`, "HTTP code, uppercase 1XX-5XX, or default"},
		{"no_final", `"103": {description: hints}`, "Responses must contain a response code"},
		{"empty", ``, "Responses must contain a response code"},
		{"info_content", `"103": {description: hints, content: {application/json: {schema: {type: string}}}}, "200": {description: ok}`, "1xx responses cannot contain content"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(fmt.Sprintf(`openapi: 3.2.0
info: {title: Rejected, version: "1"}
paths:
  /value:
    get:
      responses: {%s}
`, tc.responses))
			_, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Contains(t, diagnostics.Error(), tc.message)
		})
	}
}

func TestImportStatusDefaultReferenceClosure(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Default, version: "1"}
paths:
  /value:
    get:
      operationId: value
      responses:
        default: {$ref: '#/components/responses/Fallback'}
components:
  responses:
    Fallback:
      description: fallback
      headers:
        Trace: {$ref: '#/components/headers/Trace'}
      content:
        application/json:
          schema: {$ref: '#/components/schemas/Value'}
  headers:
    Trace:
      schema: {type: string}
  schemas:
    Value:
      type: object
      properties:
        id: {type: integer}
`)
	doc, diagnostics, _, err := AnalyzeSelected(Source{Bytes: source}, Selection{Paths: []string{"/value"}})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Len(t, doc.Components.Schemas, 1)
	require.Len(t, doc.Components.Headers, 1)
	require.Len(t, doc.Components.Responses, 1)
	require.Len(t, doc.Operations[0].Responses, 1)
	rendered, err := Render(doc, Options{})
	require.NoError(t, err)
	for _, want := range []string{`StatusFallback`, `ImportedValue`, `Trace`, `#/components/responses/Fallback`} {
		require.True(t, strings.Contains(string(rendered), want), want)
	}
	requireRenderedDesignEvaluates(t, rendered, 1)
}
