package openapiimport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestRenderPreservesByteAndBinaryFormats(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Encoded data, version: "1"}
paths:
  /data:
    get:
      operationId: data.get
      responses:
        "200":
          description: Encoded data
          content:
            application/json:
              schema: {$ref: "#/components/schemas/Payload"}
components:
  schemas:
    Payload:
      type: object
      properties:
        encoded: {type: string, format: byte}
        nullableEncoded: {type: [string, "null"], format: byte}
        raw: {type: string, format: binary}
`)

	var strictDesign []byte
	for _, allowLossy := range []bool{false, true} {
		analysis, _, err := AnalyzePartial(source, Selection{}, allowLossy)
		require.NoError(t, err)
		require.Empty(t, analysis.Blocked)
		require.Empty(t, analysis.Warnings)
		require.Empty(t, analysis.Omitted)
		require.Empty(t, analysis.OperationOmissions)
		require.Empty(t, analysis.Skipped)
		require.Len(t, analysis.Document.Operations, 1)

		design, err := Render(analysis.Document, Options{PackageName: "design"})
		require.NoError(t, err)
		require.Contains(t, string(design), `Meta("openapi:format", "byte")`)
		if allowLossy {
			require.Equal(t, strictDesign, design)
		} else {
			strictDesign = design
		}
	}

	moduleDir := requireRenderedDesignGenerates(t, strictDesign)
	generated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	parsed, err := libopenapi.NewDocument(generated)
	require.NoError(t, err)
	_, err = parsed.BuildV3Model()
	require.NoError(t, err)

	var contract map[string]any
	require.NoError(t, json.Unmarshal(generated, &contract))
	components := requireUnconstrainedMap(t, contract["components"], "components")
	schemas := requireUnconstrainedMap(t, components["schemas"], "component schemas")
	payload := requireUnconstrainedMap(t, schemas["Payload"], "Payload schema")
	properties := requireUnconstrainedMap(t, payload["properties"], "Payload properties")
	require.Equal(t, "byte", requireUnconstrainedMap(t, properties["encoded"], "encoded schema")["format"])
	require.Equal(t, "binary", requireUnconstrainedMap(t, properties["raw"], "raw schema")["format"])

	nullable := requireUnconstrainedMap(t, properties["nullableEncoded"], "nullable encoded schema")
	variants, ok := nullable["anyOf"].([]any)
	require.True(t, ok)
	require.Len(t, variants, 2)
	var stringVariant map[string]any
	for _, variant := range variants {
		candidate := requireUnconstrainedMap(t, variant, "nullable encoded variant")
		if candidate["type"] == "string" {
			stringVariant = candidate
		}
	}
	require.NotNil(t, stringVariant)
	require.Equal(t, "byte", stringVariant["format"])
}
