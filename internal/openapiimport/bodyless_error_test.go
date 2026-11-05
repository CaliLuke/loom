package openapiimport

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestRenderBodylessErrorForwardGenerates(t *testing.T) {
	document, diagnostics, err := Analyze([]byte(`openapi: 3.1.1
info: {title: Repro, version: 1.0.0}
paths:
  /items:
    post:
      operationId: createItem
      responses:
        "201": {description: Created}
        "413": {description: Payload too large}
        "429":
          description: Too many requests
          headers:
            Retry-After:
              required: true
              schema: {type: integer}
`))
	require.NoError(t, err)
	require.Empty(t, diagnostics)

	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	moduleDir := requireRenderedDesignGenerates(t, design)
	require.Contains(t, string(design), `Error("Status413", func() {`)
	require.Contains(t, string(design), `Attribute("message", String)`)
	require.NotContains(t, string(design), `Error("Status413", Empty)`)
	require.Contains(t, string(design), "\t\t\t\tBody(Empty)\n")

	generated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	parsed, err := libopenapi.NewDocument(generated)
	require.NoError(t, err)
	_, err = parsed.BuildV3Model()
	require.NoError(t, err)
	var contract map[string]any
	require.NoError(t, json.Unmarshal(generated, &contract))
	if components, ok := contract["components"].(map[string]any); ok {
		if schemas, ok := components["schemas"].(map[string]any); ok {
			require.NotContains(t, schemas, "Problem")
		}
	}
	operation := operationFromImportedSpec(t, contract, "/items", "post")
	responses := requireUnconstrainedMap(t, operation["responses"], "responses")
	tooLarge := requireUnconstrainedMap(t, responses["413"], "payload too large response")
	require.NotContains(t, tooLarge, "content")
	require.NotContains(t, tooLarge, "headers")

	tooMany := requireUnconstrainedMap(t, responses["429"], "too many requests response")
	require.NotContains(t, tooMany, "content")
	headers := requireUnconstrainedMap(t, tooMany["headers"], "too many requests headers")
	require.Len(t, headers, 1)
	require.Contains(t, headers, "Retry-After")
}
