package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportMixedArrayObjectOneOf(t *testing.T) {
	source := []byte(`openapi: 3.1.0
info: {title: Array page, version: '1'}
paths:
  /items:
    post:
      operationId: listItems
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Result'}
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Result'}
components:
  schemas:
    Result:
      oneOf:
        - {$ref: '#/components/schemas/Items'}
        - {$ref: '#/components/schemas/Page'}
    Items:
      type: array
      maxItems: 10
      items: {$ref: '#/components/schemas/Item'}
    Item:
      type: object
      additionalProperties: false
      required: [name]
      properties:
        name: {type: string, minLength: 2}
    Page:
      type: object
      required: [total]
      properties:
        total: {type: integer}
        items: {$ref: '#/components/schemas/Items'}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(rendered), "Untagged()")
	module := requireRenderedDesignGenerates(t, rendered)
	contract := readGeneratedOpenAPIContract(t, module)
	operation := operationFromImportedSpec(t, contract, "/items", "post")
	schema := operation["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	if ref, ok := schema["$ref"].(string); ok {
		schema = referencedSchema(t, contract, ref)
	}
	require.Len(t, schema["oneOf"], 2)
	require.NotContains(t, schema, "discriminator")
	for _, raw := range schema["oneOf"].([]any) {
		branch := referencedSchema(t, contract, raw.(map[string]any)["$ref"].(string))
		if branch["type"] == "array" {
			require.EqualValues(t, 10, branch["maxItems"])
			return
		}
	}
	t.Error("mixed union lost its array component")
}
