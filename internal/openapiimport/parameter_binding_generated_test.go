package openapiimport

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

const parameterBindingSource = `openapi: 3.1.1
info: {title: Bindings, version: '1'}
paths:
  /value/{ids}:
    get:
      operationId: value
      parameters:
        - {name: ids, in: path, required: true, schema: {type: array, items: {type: string}}}
        - {name: count, in: query, schema: {type: integer}}
        - {name: X-Tags, in: header, schema: {type: array, items: {$ref: '#/components/schemas/ItemName'}}}
        - {name: X-Any, in: header, schema: {}}
        - {name: session, in: cookie, schema: {type: string}}
        - name: filter
          in: query
          content:
            application/json:
              schema: %s
      responses:
        '200':
          description: ok
          headers:
            X-Values: {schema: {type: array, items: {type: string}}}
components:
  schemas:
    ItemName: {type: string}
%s
`

func TestImportedParameterBindingGenerated(t *testing.T) {
	for _, tc := range []struct {
		name, schema, components string
	}{
		{"ordinary", `{type: object, additionalProperties: {type: array, items: {type: string}}}`, ""},
		{"omitted map", `{type: object}`, ""},
		{"named alias map", `{$ref: '#/components/schemas/FilterMap'}`, `    FilterMap: {type: object, additionalProperties: {type: string}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze([]byte(fmt.Sprintf(parameterBindingSource, tc.schema, tc.components)))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			design, err := Render(document, Options{})
			require.NoError(t, err)
			module := requireRenderedDesignGenerates(t, design)
			contract := readGeneratedOpenAPIContract(t, module)
			operation := operationFromImportedSpec(t, contract, "/value/{ids}", "get")
			require.Len(t, operation["parameters"], 6)
		})
	}
}
