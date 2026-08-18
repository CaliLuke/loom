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
                format: proprietary
`)

	tests := []struct {
		name            string
		allowLossy      bool
		wantOperations  int
		wantSkipped     int
		wantWarningCode string
	}{
		{name: "strict", wantSkipped: 1},
		{name: "allow lossy", allowLossy: true, wantOperations: 1, wantWarningCode: "schema-format"},
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

func TestAnalyzePartialRetainsOneOfBranchComponents(t *testing.T) {
	source := []byte(`openapi: 3.1.0
info: {title: PartialOneOf, version: "1"}
paths:
  /items:
    get:
      operationId: getItem
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                oneOf:
                  - type: object
                    additionalProperties: false
                    required: [data]
                    properties:
                      data: {type: string}
                  - type: object
                    additionalProperties: false
                    required: [error]
                    properties:
                      error: {type: string}
`)

	analysis, _, err := AnalyzePartial(source, Selection{}, false)
	require.NoError(t, err)
	require.Equal(t, 1, analysis.TotalOperations)
	require.Equal(t, 2, analysis.TotalSchemas)
	require.Len(t, analysis.Document.Operations, 1)
	require.Empty(t, analysis.Blocked)
	require.Empty(t, analysis.Skipped)
	require.Len(t, analysis.Document.Components.Schemas, 2)
	require.Equal(t, []string{"GetItemResponseData", "GetItemResponseError"}, []string{
		analysis.Document.Components.Schemas[0].Name,
		analysis.Document.Components.Schemas[1].Name,
	})
	require.Len(t, analysis.Document.Operations[0].Responses[0].Response.Schema.OneOf, 2)
}

func TestAnalyzePartialPreservesSupportedDocumentSecurity(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info:
  title: Partial
  version: "1"
  contact: {name: Loom}
servers: [{url: https://api.example.com}]
security: [{apiKey: []}]
tags:
  - name: pets
    description: Pet operations.
paths:
  /pets:
    get:
      operationId: getPets
      tags: [pets]
      responses: {"204": {description: done}}
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
`)

	analysis, _, err := AnalyzePartial(source, Selection{}, false)
	require.NoError(t, err)
	require.Len(t, analysis.Document.Operations, 1)
	require.Empty(t, analysis.Skipped)
	require.Empty(t, analysis.Blocked)
	for _, code := range []string{"info-metadata", "servers"} {
		requireDiagnosticCode(t, analysis.Omitted, code)
	}
	require.Equal(t, "Pet operations.", analysis.Document.TagMetadata[0].Description)
	require.True(t, analysis.Document.SecurityDefined)
	require.Len(t, analysis.Document.Security, 1)
	require.Len(t, analysis.Document.Components.SecuritySchemes, 1)
}

func TestAnalyzePartialPreservesOperationLevelSecurity(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Partial, version: "1"}
security: [{apiKey: []}]
paths:
  /public:
    get:
      operationId: getPublic
      responses: {"204": {description: done}}
  /private:
    get:
      operationId: getPrivate
      security: [{apiKey: []}]
      responses: {"204": {description: done}}
components:
  securitySchemes:
    apiKey: {type: apiKey, in: header, name: X-API-Key}
`)

	analysis, _, err := AnalyzePartial(source, Selection{}, false)
	require.NoError(t, err)
	require.Len(t, analysis.Document.Operations, 2)
	require.Empty(t, analysis.Skipped)
	require.Empty(t, analysis.Blocked)
	require.Empty(t, analysis.OperationOmissions)
	private := analysis.Document.Operations[0]
	if private.Path != "/private" {
		private = analysis.Document.Operations[1]
	}
	require.True(t, private.SecurityDefined)
	require.Len(t, private.Security, 1)
}
