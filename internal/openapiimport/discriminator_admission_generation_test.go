package openapiimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscriminatorNullableAdmissionGenerates(t *testing.T) {
	for _, tc := range []struct{ name, version, form, hint string }{
		{"optional 3.1", "3.1.1", "declaration", "{propertyName: kind, x-audit: {z: 17, a: true}}"},
		{"fallback 3.2", "3.2.0", "declaration", "{propertyName: kind, defaultMapping: Cat, x-audit: {z: 17, a: true}}"},
		{"nullable occurrence", "3.2.0", "reference occurrence", "{propertyName: kind, defaultMapping: Cat}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := discriminatorNullableDocument(tc.version, tc.hint, tc.form)
			source = []byte(strings.Replace(string(source), "paths: {}", `paths:
  /pets:
    get:
      operationId: pets.get
      responses:
        '200':
          description: Pet result
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Holder'}`, 1) + `    Holder:
      type: object
      properties: {pet: {$ref: '#/components/schemas/Pet'}}
`)
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			directory := requireRenderedDesignGenerates(t, design)
			requireDiscriminatorSpecifications(t, directory)
			requireIndependentRegeneration(t, directory)
			contract := readGeneratedOpenAPIContract(t, directory)
			components := contract["components"].(map[string]any)["schemas"].(map[string]any)
			hint := components["Pet"].(map[string]any)["discriminator"].(map[string]any)
			require.Equal(t, "kind", hint["propertyName"])
			if tc.version == "3.2.0" {
				require.Equal(t, "#/components/schemas/Cat", hint["defaultMapping"])
			} else {
				require.NotContains(t, hint, "defaultMapping")
			}
			if tc.form == "declaration" {
				require.Equal(t, map[string]any{"z": float64(17), "a": true}, hint["x-audit"])
			}
			requireTypeNullAcceptance(t, source, directory, "Pet", []struct {
				value any
				valid bool
			}{
				{nil, true},
				{map[string]any{}, false},
				{map[string]any{"kind": "cat", "name": "M"}, true},
				{map[string]any{"kind": "dog", "age": 1}, true},
				{map[string]any{"name": "M"}, false},
			})
		})
	}
}
