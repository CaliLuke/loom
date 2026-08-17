package openapiimport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderPreservesSnakeCasePathParameterIdentity(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Widgets, version: "1"}
paths:
  /widgets/{asset_id}:
    get:
      operationId: getWidget
      parameters:
        - name: asset_id
          in: path
          required: true
          schema: {type: string}
      responses:
        "204": {description: found}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(rendered), `Attribute("asset_id", String)`)
	require.Contains(t, string(rendered), `GET("/widgets/{asset_id}")`)
	require.Contains(t, string(rendered), `Param("asset_id")`)
	requireRenderedDesignEvaluates(t, rendered, 1)

	contract := readGeneratedOpenAPIContract(t, requireRenderedDesignGenerates(t, rendered))
	operation := operationFromImportedSpec(t, contract, "/widgets/{asset_id}", "get")
	parameters := operation["parameters"].([]any)
	require.Equal(t, "asset_id", parameters[0].(map[string]any)["name"])
}

func TestRenderResolvesSingleReferenceAllOf(t *testing.T) {
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
                required: [status]
                properties:
                  status:
                    allOf:
                      - $ref: '#/components/schemas/Status'
components:
  schemas:
    Status:
      type: integer
      enum: [1, 2]
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(rendered), `Attribute("status", ImportedStatus)`)
	requireRenderedDesignEvaluates(t, rendered, 1)

	contract := readGeneratedOpenAPIContract(t, requireRenderedDesignGenerates(t, rendered))
	operation := operationFromImportedSpec(t, contract, "/status", "get")
	responseSchema := operation["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	if ref, ok := responseSchema["$ref"].(string); ok {
		responseSchema = referencedSchema(t, contract, ref)
	}
	statusSchema := responseSchema["properties"].(map[string]any)["status"].(map[string]any)
	if ref, ok := statusSchema["$ref"].(string); ok {
		statusSchema = referencedSchema(t, contract, ref)
	}
	require.Equal(t, "integer", statusSchema["type"])
	require.Equal(t, []any{float64(1), float64(2)}, statusSchema["enum"])
}

func TestRenderPromotesInlineAllOfArrayItemsWithLossyConsent(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Users, version: "1"}
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        "200":
          description: users
          content:
            application/json:
              schema:
                type: array
                items:
                  allOf:
                    - $ref: '#/components/schemas/UserLite'
                    - type: object
                      required: [role]
                      properties:
                        role: {type: integer}
components:
  schemas:
    UserLite:
      type: object
      required: [id]
      properties:
        id: {type: string}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "schema-allof-flattened")
	requireDiagnosticCode(t, diagnostics, "schema-inline-array-item-promoted")
	fatal, warnings := diagnostics.Classify(true)
	require.Empty(t, fatal)
	require.NotEmpty(t, warnings)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(rendered), "Extend(ImportedUserLite)")
	require.Contains(t, string(rendered), "ArrayOf(ImportedListUsersResponseItem)")
	requireRenderedDesignEvaluates(t, rendered, 1)

	contract := readGeneratedOpenAPIContract(t, requireRenderedDesignGenerates(t, rendered))
	operation := operationFromImportedSpec(t, contract, "/users", "get")
	items := operation["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)["items"].(map[string]any)
	itemSchema := referencedSchema(t, contract, items["$ref"].(string))
	properties := itemSchema["properties"].(map[string]any)
	require.Contains(t, properties, "id")
	require.Contains(t, properties, "role")
}

func TestRenderTreatsRedirectOnlyOperationAsSuccessful(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Redirects, version: "1"}
paths:
  /legacy:
    get:
      operationId: getLegacy
      responses:
        "302": {description: moved}
        "400": {description: bad request}
  /mixed:
    get:
      operationId: getMixed
      responses:
        "200": {description: current}
        "302": {description: moved}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(rendered), "Response(302")
	require.Contains(t, string(rendered), `Response("Status400", 400`)
	require.Contains(t, string(rendered), `Response("Status302", 302`)
	requireRenderedDesignEvaluates(t, rendered, 2)

	contract := readGeneratedOpenAPIContract(t, requireRenderedDesignGenerates(t, rendered))
	responses := operationFromImportedSpec(t, contract, "/legacy", "get")["responses"].(map[string]any)
	require.Contains(t, responses, "302")
	require.Contains(t, responses, "400")
	mixedResponses := operationFromImportedSpec(t, contract, "/mixed", "get")["responses"].(map[string]any)
	require.Contains(t, mixedResponses, "200")
	require.Contains(t, mixedResponses, "302")
}

func readGeneratedOpenAPIContract(t *testing.T, moduleDir string) map[string]any {
	t.Helper()
	generated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	var contract map[string]any
	require.NoError(t, json.Unmarshal(generated, &contract))
	return contract
}

func referencedSchema(t *testing.T, contract map[string]any, ref string) map[string]any {
	t.Helper()
	name := strings.TrimPrefix(ref, "#/components/schemas/")
	require.NotEqual(t, ref, name)
	return contract["components"].(map[string]any)["schemas"].(map[string]any)[name].(map[string]any)
}
