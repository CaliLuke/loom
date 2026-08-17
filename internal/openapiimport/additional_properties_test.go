package openapiimport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestRenderRoundTripsFreeFormObject(t *testing.T) {
	document, diagnostics, err := Analyze([]byte(`openapi: 3.1.1
info: {title: Opaque response, version: "1"}
paths:
  /opaque:
    get:
      operationId: opaque.get
      tags: [opaque]
      responses:
        "200":
          description: Opaque data
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/OpaqueRawOcrResults"
components:
  schemas:
    OpaqueRawOcrResults:
      type: object
      additionalProperties: true
`))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Len(t, document.Components.Schemas, 1)
	additional := document.Components.Schemas[0].Schema.AdditionalProperties
	require.NotNil(t, additional)
	require.NotNil(t, additional.Allowed)
	require.True(t, *additional.Allowed)

	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), `Type("OpaqueRawOcrResults", MapOf(String, Any), func() {`)

	moduleDir := requireRenderedDesignGenerates(t, design)
	serviceFiles, err := filepath.Glob(filepath.Join(moduleDir, "gen", "*", "service.go"))
	require.NoError(t, err)
	require.NotEmpty(t, serviceFiles)
	foundType := false
	for _, serviceFile := range serviceFiles {
		serviceSource, readErr := os.ReadFile(serviceFile)
		require.NoError(t, readErr)
		if strings.Contains(string(serviceSource), "type OpaqueRawOcrResults map[string]any") {
			foundType = true
			break
		}
	}
	require.True(t, foundType, "generated service files do not contain the free-form object type")

	generated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	parsed, err := libopenapi.NewDocument(generated)
	require.NoError(t, err)
	_, err = parsed.BuildV3Model()
	require.NoError(t, err)
	var contract map[string]any
	require.NoError(t, json.Unmarshal(generated, &contract))
	schemas := contract["components"].(map[string]any)["schemas"].(map[string]any)
	opaque := schemas["OpaqueRawOcrResults"].(map[string]any)
	require.Equal(t, "object", opaque["type"])
	require.Equal(t, true, opaque["additionalProperties"])
}
