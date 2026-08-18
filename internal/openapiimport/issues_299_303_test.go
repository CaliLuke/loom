package openapiimport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderPreservesSingleReferenceAllOfSiblingConstraints(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Status, version: "1"}
paths:
  /status:
    get:
      operationId: getStatus
      responses:
        "200":
          description: current status
          content:
            application/json:
              schema:
                type: object
                properties:
                  bounded:
                    allOf:
                      - $ref: '#/components/schemas/Status'
                    minimum: 0
                    maximum: 1
                  defaulted:
                    allOf:
                      - $ref: '#/components/schemas/Status'
                    default: 1
components:
  schemas:
    Status:
      type: integer
      enum: [0, 1]
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	design := string(rendered)
	require.Contains(t, design, `Attribute("bounded", ImportedStatus, func()`)
	require.Contains(t, design, "Minimum(0)")
	require.Contains(t, design, "Maximum(1)")
	require.Contains(t, design, `Attribute("defaulted", ImportedStatus, func()`)
	require.Contains(t, design, "Default(1)")
	requireRenderedDesignEvaluates(t, rendered, 1)

	contract := readGeneratedOpenAPIContract(t, requireRenderedDesignGenerates(t, rendered))
	operation := operationFromImportedSpec(t, contract, "/status", "get")
	responseSchema := operation["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	if ref, ok := responseSchema["$ref"].(string); ok {
		responseSchema = referencedSchema(t, contract, ref)
	}
	properties := responseSchema["properties"].(map[string]any)
	bounded := properties["bounded"].(map[string]any)
	require.Equal(t, float64(0), bounded["minimum"])
	require.Equal(t, float64(1), bounded["maximum"])
	require.Equal(t, "#/components/schemas/Status", bounded["allOf"].([]any)[0].(map[string]any)["$ref"])
	defaulted := properties["defaulted"].(map[string]any)
	require.Equal(t, float64(1), defaulted["default"])
	require.Equal(t, "#/components/schemas/Status", defaulted["allOf"].([]any)[0].(map[string]any)["$ref"])
}

func TestSingleReferenceAllOfSiblingConstraintsFailClosed(t *testing.T) {
	tests := map[string]struct {
		sibling string
		code    string
	}{
		"incompatible default": {sibling: `default: invalid`, code: "schema-keyword"},
		"structural type":      {sibling: `type: integer`, code: "schema-composition"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			source := []byte(`openapi: 3.1.1
info: {title: Status, version: "1"}
paths:
  /status:
    get:
      operationId: getStatus
      responses:
        "200":
          description: current status
          content:
            application/json:
              schema:
                allOf:
                  - $ref: '#/components/schemas/Status'
                ` + test.sibling + `
components:
  schemas:
    Status: {type: integer}
`)

			_, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			requireDiagnosticCode(t, diagnostics, test.code)
		})
	}
}
