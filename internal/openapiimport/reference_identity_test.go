package openapiimport

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeEncodedComponentReference(t *testing.T) {
	source, err := os.ReadFile("testdata/encoded_component_reference.yaml")
	require.NoError(t, err)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	_, err = Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
}

func TestEncodedReferenceSelection(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Reference selection, version: '1'}
paths:
  /value:
    get:
      operationId: value
      parameters:
        - {$ref: '#/components/parameters/%54ags'}
      responses:
        '200':
          description: done
          content:
            application/json:
              schema: {type: string}
              examples:
                chosen: {$ref: '#/components/examples/%53ample'}
components:
  parameters:
    Tags:
      name: X-Tags
      in: header
      schema: {type: array, items: {$ref: '#/components/schemas/%49temName'}}
  schemas:
    ItemName: {type: string}
  examples:
    Sample: {value: red}
`)
	document, diagnostics, _, err := AnalyzeSelected(source, Selection{Paths: []string{"/value"}})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Len(t, document.Components.Schemas, 1)
	require.Equal(t, "ItemName", document.Components.Schemas[0].Name)
	require.Len(t, document.Components.Parameters, 1)
	require.Equal(t, "Tags", document.Components.Parameters[0].Name)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(rendered), "red")
}

func TestEncodedReferenceGeneratedContract(t *testing.T) {
	source, err := os.ReadFile("testdata/encoded_component_reference.yaml")
	require.NoError(t, err)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	module := requireRenderedDesignGenerates(t, rendered)
	for _, name := range []string{"openapi.json", "openapi.yaml"} {
		data, err := os.ReadFile(filepath.Join(module, "gen", "http", name))
		require.NoError(t, err)
		parsed, err := libopenapi.NewDocument(data)
		require.NoError(t, err)
		_, err = parsed.BuildV3Model()
		require.NoError(t, err)
	}
	contract := readGeneratedOpenAPIContract(t, module)
	operation := operationFromImportedSpec(t, contract, "/value", "get")
	parameter := operation["parameters"].([]any)[0].(map[string]any)
	item := parameter["schema"].(map[string]any)["items"].(map[string]any)
	require.Equal(t, "#/components/schemas/ItemName", item["$ref"])
	require.Contains(t, contract["components"].(map[string]any)["schemas"], "ItemName")
}
