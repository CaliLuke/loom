package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnalyzePartialSkipsOnlyBlockedOperations(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Partial, version: "1"}
paths:
  /good:
    get:
      operationId: getGood
      responses:
        "200":
          description: done
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Good'}
  /bad:
    post:
      operationId: postBad
      callbacks: {unsupported: {}}
      responses:
        "200":
          description: done
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Bad'}
components:
  schemas:
    Good: {type: string}
    Bad:
      oneOf:
        - {type: string}
        - {type: integer}
`)

	analysis, _, err := AnalyzePartial(source, Selection{}, false)
	require.NoError(t, err)
	require.Equal(t, 2, analysis.TotalOperations)
	require.Equal(t, 2, analysis.TotalSchemas)
	require.Len(t, analysis.Document.Operations, 1)
	require.Equal(t, "/good", analysis.Document.Operations[0].Path)
	require.Len(t, analysis.Document.Components.Schemas, 1)
	require.Equal(t, "Good", analysis.Document.Components.Schemas[0].Name)
	require.Len(t, analysis.Skipped, 1)
	require.Equal(t, "POST", analysis.Skipped[0].Method)
	require.Equal(t, "/bad", analysis.Skipped[0].Path)
	requireDiagnosticCode(t, analysis.Skipped[0].Diagnostics, "callbacks")
	requireDiagnosticCode(t, analysis.Skipped[0].Diagnostics, "schema-composition")
}

func TestAnalyzePartialHonorsLossyPolicy(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Partial, version: "1"}
paths:
  /example:
    get:
      responses:
        "200":
          description: done
          content:
            application/json:
              schema:
                type: string
                example: value
`)

	tests := []struct {
		name            string
		allowLossy      bool
		wantOperations  int
		wantSkipped     int
		wantWarningCode string
	}{
		{name: "strict", wantSkipped: 1},
		{name: "allow lossy", allowLossy: true, wantOperations: 1, wantWarningCode: "examples"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			analysis, _, err := AnalyzePartial(source, Selection{}, test.allowLossy)
			require.NoError(t, err)
			require.Len(t, analysis.Document.Operations, test.wantOperations)
			require.Len(t, analysis.Skipped, test.wantSkipped)
			if test.wantWarningCode != "" {
				requireDiagnosticCode(t, analysis.Warnings, test.wantWarningCode)
			}
		})
	}
}
