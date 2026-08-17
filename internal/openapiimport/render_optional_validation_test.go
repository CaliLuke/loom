package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderGeneratesCompilingNestedOptionalResponseValidation(t *testing.T) {
	document, diagnostics, err := Analyze([]byte(`openapi: 3.1.0
info:
  title: Optional Validation
  version: "1"
paths:
  /scores:
    post:
      operationId: addScore
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Envelope"
components:
  schemas:
    Envelope:
      type: object
      properties:
        scores:
          $ref: "#/components/schemas/Scores"
    Scores:
      type: object
      required:
        - requiredCode
        - requiredConfidence
        - requiredDetails
        - requiredLabels
      properties:
        confidence:
          type: number
          minimum: 0
          maximum: 1
        code:
          type: string
          minLength: 2
          maxLength: 8
        details:
          $ref: "#/components/schemas/Details"
        labels:
          type: array
          items:
            type: string
            minLength: 1
        requiredCode:
          type: string
          minLength: 2
          maxLength: 8
        requiredConfidence:
          type: number
          minimum: 0
          maximum: 1
        requiredDetails:
          $ref: "#/components/schemas/Details"
        requiredLabels:
          type: array
          items:
            type: string
            minLength: 1
    Details:
      type: object
      properties:
        note:
          type: string
          minLength: 1
`))
	require.NoError(t, err)
	require.Empty(t, diagnostics)

	source, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, source, 1)
	requireRenderedDesignGenerates(t, source)
}
