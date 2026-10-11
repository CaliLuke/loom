package openapiimport

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParameterReferenceDescriptions(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.1.1", "3.2.0"} {
		t.Run(version, func(t *testing.T) {
			source := []byte(fmt.Sprintf(`openapi: %s
info: {title: Overlay, version: "1"}
paths:
  /override:
    get:
      operationId: override
      parameters:
        - $ref: '#/components/parameters/Limit'
          description: Local description.
      responses: {"204": {description: done}}
  /empty:
    get:
      operationId: empty
      parameters:
        - $ref: '#/components/parameters/Limit'
          description: ""
      responses: {"204": {description: done}}
  /inherit:
    get:
      operationId: inherit
      parameters:
        - $ref: '#/components/parameters/Limit'
      responses: {"204": {description: done}}
components:
  parameters:
    Limit:
      name: limit
      in: query
      description: Component description.
      schema: {type: integer}
`, version))
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			want := map[string]string{"/override": "Local description.", "/empty": "", "/inherit": "Component description."}
			if version == "3.0.3" {
				want["/override"] = "Component description."
				want["/empty"] = "Component description."
			}
			for _, operation := range document.Operations {
				parameter, _, err := resolveParameterReference(operation.Parameters[0], document.Components)
				require.NoError(t, err)
				require.Equal(t, want[operation.Path], parameter.Description)
			}
			require.Equal(t, "Component description.", document.Components.Parameters[0].Parameter.Description)
			rendered, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			contract := readGeneratedOpenAPIContract(t, requireRenderedDesignGenerates(t, rendered))
			components := contract["components"].(map[string]any)["parameters"].(map[string]any)
			for path, description := range want {
				operation := operationFromImportedSpec(t, contract, path, "get")
				parameter := operation["parameters"].([]any)[0].(map[string]any)
				if ref, ok := parameter["$ref"].(string); ok {
					name, err := localComponentReferenceName(ref, "#/components/parameters/")
					require.NoError(t, err)
					parameter = components[name].(map[string]any)
				}
				got, _ := parameter["description"].(string)
				require.Equal(t, description, got, path)
			}
		})
	}
}
