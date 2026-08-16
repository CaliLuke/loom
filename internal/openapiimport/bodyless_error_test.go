package openapiimport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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
	require.Contains(t, string(design), `Error("Status413")`)
	require.NotContains(t, string(design), `Error("Status413", Empty)`)
	require.Contains(t, string(design), "\t\t\t\tBody(Empty)\n")

	generated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	var contract map[string]any
	require.NoError(t, json.Unmarshal(generated, &contract))
	if components, ok := contract["components"].(map[string]any); ok {
		if schemas, ok := components["schemas"].(map[string]any); ok {
			require.NotContains(t, schemas, "Problem")
		}
	}
	for _, test := range []struct {
		path   string
		method string
		status string
	}{
		{path: "/items", method: "post", status: "413"},
		{path: "/items", method: "post", status: "429"},
	} {
		operation := operationFromImportedSpec(t, contract, test.path, test.method)
		responses := requireUnconstrainedMap(t, operation["responses"], "responses")
		response := requireUnconstrainedMap(t, responses[test.status], "bodyless error response")
		require.NotContains(t, response, "content")
	}
}
