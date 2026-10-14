package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportedYAMLComponentIndex(t *testing.T) {
	const prefix = `openapi: 3.1.1
info: {title: Index, version: '1'}
x-schema: &schema {type: integer, minimum: 5, maximum: 10}
x-schemas: &schemas {Bound: *schema}
x-components: &components {schemas: *schemas}
x-sources: &sources [*components]
paths:
  /bound:
    get:
      operationId: readBound
      responses:
        '200':
          description: Bound
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Bound'}
`
	var explicit []byte
	for _, tc := range []struct{ name, components string }{
		{"plain", "components: {schemas: {Bound: {type: integer, minimum: 5, maximum: 10}}}\n"},
		{"components merge", "components: {<<: *sources}\n"},
		{"schemas merge", "components: {schemas: {<<: *schemas}}\n"},
		{"components alias", "components: *components\n"},
		{"schemas alias", "components: {schemas: *schemas}\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze([]byte(prefix + tc.components))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			require.Len(t, document.Components.Schemas, 1)
			schema := document.Components.Schemas[0].Schema
			require.Equal(t, "5", schema.Minimum.String())
			require.Equal(t, "10", schema.Maximum.String())
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			if explicit == nil {
				explicit = design
			}
			require.Equal(t, string(explicit), string(design), "YAML composition must preserve the referenced contract")
		})
	}
}
