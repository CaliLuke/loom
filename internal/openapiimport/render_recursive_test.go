package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderRecursiveComponentsEvaluate(t *testing.T) {
	cases := []struct {
		name    string
		schemas string
	}{
		{"acyclic", `Node: {type: object, properties: {name: {type: string}}}`},
		{"self_array", `Node:
      type: object
      properties:
        name: {type: string}
        children: {type: array, items: {$ref: '#/components/schemas/Node'}}`},
		{"mutual_map_array", `Node:
      type: object
      properties:
        branches: {type: object, additionalProperties: {$ref: '#/components/schemas/Branch'}}
    Branch:
      type: object
      properties:
        children: {type: array, items: {$ref: '#/components/schemas/Node'}}`},
		{"mixed_inheritance", `Node:
      allOf:
        - {$ref: '#/components/schemas/Base'}
        - type: object
          properties: {name: {type: string}}
    Base:
      type: object
      properties:
        child: {$ref: '#/components/schemas/Node'}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(`openapi: 3.1.1
info: {title: Recursive, version: "1"}
paths:
  /nodes:
    get:
      operationId: getNodes
      responses:
        "200":
          description: Node
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Node'}
components:
  schemas:
    ` + tc.schemas + "\n")
			doc, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			design, err := Render(doc, Options{PackageName: "design"})
			require.NoError(t, err)
			requireRenderedDesignEvaluates(t, design, 1)
			requireRecursiveDesignGenerates(t, design, tc.name)
		})
	}
}
