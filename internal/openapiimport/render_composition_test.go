package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderSpringAllOfInheritance(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Documents, version: "1"}
paths:
  /attempts/{id}:
    get:
      operationId: getAttempt
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      responses:
        "200":
          description: attempt
          content:
            application/json:
              schema: {$ref: '#/components/schemas/AsylumSeekerDocumentAttempt'}
components:
  schemas:
    AdditionalDocumentAttempt:
      type: object
      required: [id]
      properties:
        id: {type: string}
    AsylumSeekerDocumentAttempt:
      allOf:
        - {$ref: '#/components/schemas/AdditionalDocumentAttempt'}
        - type: object
          required: [firstName]
          properties:
            countryOfOrigin: {type: string}
            firstName: {type: string}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "schema-allof-flattened")
	requireNoDiagnosticCode(t, diagnostics, "schema")
	requireNoDiagnosticCode(t, diagnostics, "schema-composition")
	fatal, warnings := diagnostics.Classify(true)
	require.Empty(t, fatal)
	require.NotEmpty(t, warnings)

	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(rendered), `Extend(ImportedAdditionalDocumentAttempt)`)
	require.Contains(t, string(rendered), `Attribute("firstName", String)`)
	require.Contains(t, string(rendered), `Required("firstName")`)
	requireRenderedDesignEvaluates(t, rendered, 1)
}

func requireNoDiagnosticCode(t *testing.T, diagnostics Diagnostics, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		require.NotEqual(t, code, diagnostic.Code, diagnostics.Error())
	}
}
