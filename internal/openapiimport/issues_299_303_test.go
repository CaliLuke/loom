package openapiimport

import (
	"strings"
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

func TestRenderPreservesNullableUnconstrainedSchema(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Values, version: "1"}
paths:
  /value:
    get:
      operationId: getValue
      responses:
        "200":
          description: value
          content:
            application/json:
              schema:
                type: object
                properties:
                  value:
                    anyOf:
                      - {}
                      - type: "null"
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	property := document.Operations[0].Responses[0].Response.Schema.Properties[0].Schema
	require.True(t, property.Unconstrained)
	require.False(t, property.Nullable)

	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(rendered), `Attribute("value", Any, func()`)
	requireRenderedDesignEvaluates(t, rendered, 1)

	contract := readGeneratedOpenAPIContract(t, requireRenderedDesignGenerates(t, rendered))
	operation := operationFromImportedSpec(t, contract, "/value", "get")
	responseSchema := operation["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	if ref, ok := responseSchema["$ref"].(string); ok {
		responseSchema = referencedSchema(t, contract, ref)
	}
	value := responseSchema["properties"].(map[string]any)["value"].(map[string]any)
	require.Empty(t, value)
}

func TestRenderPreservesNullableReferenceOneOf(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Widgets, version: "1"}
paths:
  /widget:
    get:
      operationId: getWidget
      responses:
        "200":
          description: current widget
          content:
            application/json:
              schema: {$ref: '#/components/schemas/NullableWidget'}
components:
  schemas:
    Widget:
      type: object
      required: [name]
      properties:
        name: {type: string}
    NullableWidget:
      oneOf:
        - $ref: '#/components/schemas/Widget'
        - type: "null"
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Len(t, document.Components.Schemas, 2)
	var nullable *Schema
	for _, named := range document.Components.Schemas {
		if named.Name == "NullableWidget" {
			nullable = named.Schema
		}
	}
	require.NotNil(t, nullable)
	require.Equal(t, "#/components/schemas/Widget", nullable.Ref)
	require.True(t, nullable.Nullable)

	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	design := string(rendered)
	require.Contains(t, design, `var ImportedNullableWidget = Type("NullableWidget", ImportedWidget, func() {`)
	require.Contains(t, design, "Nullable()")
	requireRenderedDesignEvaluates(t, rendered, 1)

	contract := readGeneratedOpenAPIContract(t, requireRenderedDesignGenerates(t, rendered))
	nullableContract := contract["components"].(map[string]any)["schemas"].(map[string]any)["NullableWidget"].(map[string]any)
	variants := nullableContract["anyOf"].([]any)
	require.Len(t, variants, 2)
	require.Equal(t, "#/components/schemas/Widget", variants[0].(map[string]any)["$ref"])
	require.Equal(t, "null", variants[1].(map[string]any)["type"])
}

func TestNullableReferenceOneOfRejectsNullableTarget(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Widgets, version: "1"}
paths: {}
components:
  schemas:
    Widget:
      type: [object, "null"]
      properties:
        name: {type: string}
    NullableWidget:
      oneOf:
        - $ref: '#/components/schemas/Widget'
        - type: "null"
`)

	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "schema-composition")
}

func TestRenderPreservesDynamicFormMapRequestBody(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Config, version: "1"}
paths:
  /config:
    patch:
      operationId: updateConfig
      requestBody:
        content:
          application/x-www-form-urlencoded:
            schema:
              type: object
              additionalProperties: {type: string}
      responses:
        "200": {description: OK}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	design := string(rendered)
	require.Contains(t, design, "Payload(MapOf(String, String))")
	require.Contains(t, design, "FormRequest()")
	require.Contains(t, design, "OptionalRequestBody()")
	requireRenderedDesignEvaluates(t, rendered, 1)

	contract := readGeneratedOpenAPIContract(t, requireRenderedDesignGenerates(t, rendered))
	operation := operationFromImportedSpec(t, contract, "/config", "patch")
	schema := operation["requestBody"].(map[string]any)["content"].(map[string]any)["application/x-www-form-urlencoded"].(map[string]any)["schema"].(map[string]any)
	require.NotEqual(t, true, operation["requestBody"].(map[string]any)["required"])
	require.Equal(t, "object", schema["type"])
	require.Equal(t, "string", schema["additionalProperties"].(map[string]any)["type"])
}

func TestRenderPlansDynamicFormMapRequestBodies(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Config Plans, version: "1"}
paths:
  /required:
    patch:
      operationId: updateRequired
      requestBody:
        required: true
        content:
          application/x-www-form-urlencoded:
            schema: {$ref: '#/components/schemas/Config'}
      responses: {"200": {description: OK}}
  /envelope/{id}:
    patch:
      operationId: updateEnvelope
      security: [{ApiKey: []}]
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
        - {name: body2, in: header, schema: {type: string}}
      requestBody:
        required: true
        content:
          application/x-www-form-urlencoded:
            schema: {$ref: '#/components/schemas/Config'}
      responses: {"200": {description: OK}}
  /multi/{id}:
    patch:
      operationId: updateMulti
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      requestBody:
        content:
          application/json: &configBody
            schema: {$ref: '#/components/schemas/Config'}
          application/x-www-form-urlencoded: *configBody
          multipart/form-data: *configBody
      responses: {"200": {description: OK}}
  /multipart-required:
    patch:
      operationId: updateMultipartRequired
      requestBody:
        required: true
        content:
          multipart/form-data:
            schema: {$ref: '#/components/schemas/Config'}
      responses: {"200": {description: OK}}
  /multipart-optional:
    patch:
      operationId: updateMultipartOptional
      requestBody:
        content:
          multipart/form-data:
            schema: {$ref: '#/components/schemas/Config'}
      responses: {"200": {description: OK}}
components:
  schemas:
    Config:
      type: object
      additionalProperties: {type: string}
  securitySchemes:
    ApiKey: {type: apiKey, in: header, name: X-API-Key}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	design := string(rendered)
	for _, expected := range []string{
		`var ImportedConfig = Type("Config", MapOf(String, String), func() {`,
		"Payload(ImportedConfig)",
		`Attribute("body3", ImportedConfig)`,
		`Required("id", "body3")`,
		`Body("body3")`,
		`OpenAPIRequestBodyTypes(ImportedConfig, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, false)`,
		`OpenAPIRequestBodyTypes(ImportedConfig, []string{"multipart/form-data"}, false)`,
		"MultipartRequest()",
	} {
		require.Contains(t, design, expected)
	}
	requireRenderedDesignEvaluates(t, rendered, 5)

	contract := readGeneratedOpenAPIContract(t, requireRenderedDesignGenerates(t, rendered))
	for _, path := range []string{"/required", "/envelope/{id}", "/multi/{id}", "/multipart-required", "/multipart-optional"} {
		operation := operationFromImportedSpec(t, contract, path, "patch")
		require.NotNil(t, operation["requestBody"], path)
	}
	required := operationFromImportedSpec(t, contract, "/required", "patch")["requestBody"].(map[string]any)
	require.Equal(t, true, required["required"])
	require.Equal(t, "#/components/schemas/Config", required["content"].(map[string]any)["application/x-www-form-urlencoded"].(map[string]any)["schema"].(map[string]any)["$ref"])
	envelope := operationFromImportedSpec(t, contract, "/envelope/{id}", "patch")["requestBody"].(map[string]any)
	require.Equal(t, true, envelope["required"])
	require.Equal(t, "#/components/schemas/Config", envelope["content"].(map[string]any)["application/x-www-form-urlencoded"].(map[string]any)["schema"].(map[string]any)["$ref"])
	multi := operationFromImportedSpec(t, contract, "/multi/{id}", "patch")["requestBody"].(map[string]any)
	require.ElementsMatch(t, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, mapKeysAny(multi["content"].(map[string]any)))
	optionalMultipart := operationFromImportedSpec(t, contract, "/multipart-optional", "patch")["requestBody"].(map[string]any)
	require.NotEqual(t, true, optionalMultipart["required"])
}

func mapKeysAny(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func TestDynamicFormMapEnvelopeRejectsGeneratedBodyLocalCollision(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Collision, version: "1"}
paths:
  /config/{body}:
    patch:
      operationId: updateConfig
      parameters:
        - {name: body, in: path, required: true, schema: {type: string}}
      requestBody:
        required: true
        content:
          application/x-www-form-urlencoded:
            schema: {type: object, additionalProperties: {type: string}}
      responses: {"200": {description: OK}}
`)

	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "render-plan")
	require.Contains(t, diagnostics.Error(), `request body parameter "body" in "path" collides with the generated request body local`)
}

func TestOptionalFormObjectWithRequiredMembersUsesRawBody(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Optional Form, version: "1"}
paths:
  /config:
    patch:
      operationId: updateConfig
      requestBody:
        content:
          application/x-www-form-urlencoded:
            schema:
              type: object
              required: [name]
              properties:
                name: {type: string}
      responses: {"200": {description: OK}}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	method := string(rendered)
	start := strings.Index(method, `Method("UpdateConfig", func() {`)
	require.NotEqual(t, -1, start)
	method = method[start:]
	require.Contains(t, method, "SkipRequestBodyEncodeDecode()")
	require.Contains(t, method, `OpenAPIRequestBodyTypes(func() {`)
	require.NotContains(t, method, "FormRequest()")
	requireRenderedDesignEvaluates(t, rendered, 1)
}
