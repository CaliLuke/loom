package openapiimport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderClonesSharedSchemaUsedByDifferentErrors(t *testing.T) {
	tests := []struct {
		name       string
		schemas    string
		wantClones bool
	}{
		{
			name: "object",
			schemas: `    ErrorResponse:
      type: object
      properties:
        message:
          type: string`,
			wantClones: true,
		},
		{
			name: "map",
			schemas: `    ErrorResponse:
      type: object
      additionalProperties:
        type: string`,
		},
		{
			name: "alias to map",
			schemas: `    ErrorResponse:
      $ref: "#/components/schemas/ErrorValues"
    ErrorValues:
      type: object
      additionalProperties:
        type: string`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, diagnostics, err := Analyze([]byte(fmt.Sprintf(`openapi: 3.1.0
info:
  title: Shared Errors
  version: "1"
paths:
  /items:
    get:
      operationId: getItems
      responses:
        "200":
          description: OK
        "401":
          description: Unauthorized
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
        "403":
          description: Forbidden
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
components:
  schemas:
%s
`, test.schemas)))
			require.NoError(t, err)
			require.Empty(t, diagnostics)

			source, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			design := string(source)
			if test.wantClones {
				require.Contains(t, design, `Error("Status401", func() {`)
				require.Contains(t, design, `Error("Status403", func() {`)
				require.Equal(t, 2, strings.Count(design, `Extend(ImportedErrorResponse)`))
			} else {
				require.NotContains(t, design, `Extend(ImportedErrorResponse)`)
			}
			require.Equal(t, 2, strings.Count(design, `OpenAPIBody(ImportedErrorResponse)`))
			require.NotContains(t, design, `Body("body")`)
			requireRenderedDesignEvaluates(t, source, 1)
			requireRenderedDesignGenerates(t, source)
		})
	}
}
