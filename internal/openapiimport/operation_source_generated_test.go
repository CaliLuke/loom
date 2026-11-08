package openapiimport

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestOperationSourceGenerated(t *testing.T) {
	source := resolverFiles(t, []byte(`openapi: 3.2.0
info: {title: Authored operations, version: '1'}
paths:
  /cache:
    get:
      operationId: readCache
      security: [{'auth.yaml#/components/securitySchemes/K': []}]
      responses: {'204': {description: OK}}
    additionalOperations:
      PURGE:
        operationId: purgeCache
        security: [{'auth.yaml#/components/securitySchemes/K': []}]
        requestBody:
          required: true
          content:
            application/json:
              schema: {$ref: 'types.yaml#/components/schemas/Envelope/oneOf/0'}
        responses: {'204': {description: OK}}
  /public:
    get:
      operationId: readPublic
      security: []
      responses: {'204': {description: OK}}
`), map[string]string{
		"auth.yaml":  "openapi: 3.2.0\ninfo: {title: Auth, version: '1'}\npaths: {}\ncomponents:\n  securitySchemes:\n    K: {type: apiKey, in: header, name: X-Key}\n",
		"types.yaml": "openapi: 3.2.0\ninfo: {title: Types, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Envelope:\n      oneOf:\n        - {type: object, properties: {id: {type: string}}, required: [id]}\n        - {type: boolean}\n",
	})
	document, diagnostics, _, err := AnalyzeSelected(source, Selection{})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	moduleDir := requireRenderedDesignGenerates(t, rendered)
	spec, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	parsed, err := libopenapi.NewDocument(spec)
	require.NoError(t, err)
	model, err := parsed.BuildV3Model()
	require.NoError(t, err)
	path := model.Model.Paths.PathItems.GetOrZero("/cache")
	require.NotNil(t, path)
	purge := path.AdditionalOperations.GetOrZero("PURGE")
	require.NotNil(t, purge)
	require.NotNil(t, path.Get)
	require.Len(t, path.Get.Security, 1)
	require.Len(t, purge.Security, 1)
	require.Equal(t, 1, path.Get.Security[0].Requirements.Len())
	require.Equal(t, 1, purge.Security[0].Requirements.Len())
	for _, requirements := range []string{path.Get.Security[0].Requirements.First().Key(), purge.Security[0].Requirements.First().Key()} {
		require.Equal(t, "K", requirements)
	}
	require.Empty(t, model.Model.Paths.PathItems.GetOrZero("/public").Get.Security)
	scheme := model.Model.Components.SecuritySchemes.GetOrZero("K")
	require.NotNil(t, scheme)
	require.Equal(t, "apiKey", scheme.Type)
	require.Equal(t, "X-Key", scheme.Name)
	schema := purge.RequestBody.Content.GetOrZero("application/json").Schema.Schema()
	require.NotNil(t, schema.Properties.GetOrZero("id"))
	require.Equal(t, []string{"id"}, schema.Required)
}
