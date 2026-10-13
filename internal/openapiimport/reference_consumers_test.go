package openapiimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodedReferenceConsumerEquivalence(t *testing.T) {
	for _, tc := range []struct {
		name, operation, schemas string
	}{
		{"direct header", `get:
      parameters:
        - name: X-Value
          in: header
          schema: {$ref: '#/components/schemas/ItemName'}
      responses: {'204': {description: done}}`, `ItemName: {type: string}`},
		{"body plus query", `post:
      parameters:
        - {name: q, in: query, schema: {type: string}}
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/ItemName'}
      responses: {'204': {description: done}}`, `ItemName: {type: object, properties: {value: {type: string}}}`},
		{"nullable field", `get:
      responses:
        '200':
          description: done
          content:
            application/json:
              schema: {type: object, properties: {value: {$ref: '#/components/schemas/ItemName'}}}`, `ItemName: {type: [string, 'null']}`},
		{"form body", `post:
      requestBody:
        required: true
        content:
          application/x-www-form-urlencoded:
            schema: {$ref: '#/components/schemas/ItemName'}
      responses: {'204': {description: done}}`, `ItemName: {type: object, properties: {value: {type: string}}}`},
		{"nullable union", `get:
      responses:
        '200':
          description: done
          content:
            application/json:
              schema: {oneOf: [{$ref: '#/components/schemas/ItemName'}, {type: 'null'}]}`, `ItemName: {type: object, properties: {value: {type: string}}}`},
		{"single allOf", `get:
      responses:
        '200':
          description: done
          content:
            application/json:
              schema: {allOf: [{$ref: '#/components/schemas/ItemName'}]}`, `ItemName: {type: string}`},
		{"untagged union", `get:
      responses:
        '200':
          description: done
          content:
            application/json:
              schema: {oneOf: [{$ref: '#/components/schemas/ItemName'}, {$ref: '#/components/schemas/Other'}]}`, `ItemName: {type: object, properties: {value: {type: string}}, required: [value]}
    Other: {type: array, items: {type: integer}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := "openapi: 3.1.1\ninfo: {title: Identity, version: '1'}\npaths:\n  /value:\n    " + tc.operation + "\ncomponents:\n  schemas:\n    " + tc.schemas + "\n"
			ordinary, d, err := Analyze([]byte(source))
			require.NoError(t, err)
			require.Empty(t, d)
			want, err := Render(ordinary, Options{PackageName: "design"})
			require.NoError(t, err)
			for _, ref := range []string{"#/components/schemas/%49temName", "#%2Fcomponents%2Fschemas%2FItemName"} {
				encoded, d, err := Analyze([]byte(strings.ReplaceAll(source, "#/components/schemas/ItemName", ref)))
				require.NoError(t, err)
				require.Empty(t, d)
				got, err := Render(encoded, Options{PackageName: "design"})
				require.NoError(t, err)
				require.Equal(t, string(want), string(got))
			}
		})
	}
}
