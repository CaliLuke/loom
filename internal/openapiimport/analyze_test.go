package openapiimport

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnalyzeJSONYAMLParity(t *testing.T) {
	yamlSource := readFixture(t, "supported.yaml")
	jsonSource := readFixture(t, "supported.json")

	yamlDocument, yamlDiagnostics, err := Analyze(yamlSource)
	require.NoError(t, err)
	require.Empty(t, yamlDiagnostics)
	jsonDocument, jsonDiagnostics, err := Analyze(jsonSource)
	require.NoError(t, err)
	require.Empty(t, jsonDiagnostics)

	require.Equal(t, yamlDocument, jsonDocument)
	require.Equal(t, "3.1.1", yamlDocument.OpenAPIVersion)
	require.Equal(t, "Pet API", yamlDocument.Title)
	require.Len(t, yamlDocument.Components.Schemas, 4)
	require.Len(t, yamlDocument.Components.Parameters, 1)
	require.Len(t, yamlDocument.Operations, 3)
	require.Equal(t, "LegacyGet", yamlDocument.Operations[0].GoName)
	require.Equal(t, "PetsCreate", yamlDocument.Operations[1].GoName)
	require.Equal(t, "#/components/schemas/NewPet", yamlDocument.Operations[1].RequestBody.Schema.Ref)
	require.Equal(t, "PetsGet", yamlDocument.Operations[2].GoName)
	require.Len(t, yamlDocument.Operations[2].Parameters, 3)
	require.Equal(t, "#/components/parameters/PetID", yamlDocument.Operations[2].Parameters[0].Ref)
	require.Equal(t, "#/components/schemas/Pet", yamlDocument.Operations[2].Responses[0].Response.Schema.Ref)
}

func TestAnalyzeSupportsAPIKeySecurityContracts(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Secured API, version: "1"}
security:
  - HeaderKey: []
paths:
  /inherited:
    get:
      operationId: inherited
      responses: {"204": {description: done}}
  /public:
    get:
      operationId: public
      security: []
      responses: {"204": {description: done}}
  /optional:
    get:
      operationId: optional
      security:
        - CookieKey: []
        - {}
      responses: {"204": {description: done}}
components:
  securitySchemes:
    HeaderKey:
      type: apiKey
      in: header
      name: X-API-Key
      description: Header credential.
    CookieKey:
      type: apiKey
      in: cookie
      name: session-id
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.NotNil(t, document)
	require.Len(t, document.Operations, 3)
}

func TestAnalyzeRejectsUnsupportedSecuritySchemeKinds(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Unsupported security, version: "1"}
security: [{OIDC: []}]
paths:
  /items:
    get:
      operationId: listItems
      responses: {"204": {description: done}}
components:
  securitySchemes:
    OIDC:
      type: openIdConnect
      openIdConnectUrl: https://example.com/.well-known/openid-configuration
`)

	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "security-scheme")
}

func TestAnalyzeAssignsDeterministicCollisionNames(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Collisions, version: "1"}
paths:
  /b:
    get:
      operationId: get_pet
      responses: {"204": {description: done}}
  /a:
    get:
      operationId: get-pet
      responses: {"204": {description: done}}
components:
  schemas:
    pet_id: {type: string}
    pet-id: {type: integer, format: int64}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Equal(t, []string{"GetPet", "GetPet2"}, []string{
		document.Operations[0].GoName,
		document.Operations[1].GoName,
	})
	require.Equal(t, []string{"PetID", "PetID2"}, []string{
		document.Components.Schemas[0].GoName,
		document.Components.Schemas[1].GoName,
	})
}

func TestAnalyzeNormalizesOperationNames(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Operation names, version: "1"}
paths:
  /omni/get/device-info:
    get:
      operationId: getdeviceinfo
      responses: {"204": {description: done}}
  /b2b/device-info:
    get:
      operationId: b2bDeviceInfo
      responses: {"204": {description: done}}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	tests := []struct {
		name            string
		operation       Operation
		wantGoName      string
		wantOperationID string
	}{
		{
			name:            "B2B initialism",
			operation:       document.Operations[0],
			wantGoName:      "B2BDeviceInfo",
			wantOperationID: "b2bDeviceInfo",
		},
		{
			name:            "path word hints",
			operation:       document.Operations[1],
			wantGoName:      "GetDeviceInfo",
			wantOperationID: "getdeviceinfo",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.wantGoName, test.operation.GoName)
			require.Equal(t, test.wantOperationID, test.operation.OperationID)
		})
	}
}

func TestAnalyzeAggregatesAndSortsUnsupportedFeatures(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Unsupported, version: "1"}
servers: [{url: https://example.com}]
security: [{token: []}]
paths:
  /pets:
    get:
      callbacks:
        update:
          '{$request.body#/callback}':
            post:
              responses: {"204": {description: done}}
      responses:
        default: {description: fallback}
components:
  securitySchemes:
    token: {type: mutualTLS}
  schemas:
    Pet:
      oneOf:
        - {type: string}
        - {type: integer}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.NotNil(t, document)
	require.GreaterOrEqual(t, len(diagnostics), 5)
	for i := 1; i < len(diagnostics); i++ {
		require.LessOrEqual(t, diagnostics[i-1].Path, diagnostics[i].Path)
	}
	requireDiagnosticCode(t, diagnostics, "callbacks")
	requireDiagnosticCode(t, diagnostics, "default-response")
	requireDiagnosticCode(t, diagnostics, "schema-composition")
	requireDiagnosticCode(t, diagnostics, "security-requirement")
	requireDiagnosticCode(t, diagnostics, "security-scheme")
	requireDiagnosticCode(t, diagnostics, "servers")
	requireNoDiagnosticCode(t, diagnostics, "schema")
}

func TestAnalyzeRejectsUnsupportedVersions(t *testing.T) {
	tests := map[string]string{
		"swagger": `swagger: "2.0"
info: {title: Old, version: "1"}
paths: {}`,
		"future": `openapi: 3.3.0
info: {title: Future, version: "1"}
paths: {}`,
	}
	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			document, diagnostics, err := Analyze([]byte(source))
			require.Error(t, err)
			require.ErrorIs(t, err, ErrUnsupportedVersion)
			require.Nil(t, document)
			require.Empty(t, diagnostics)
		})
	}
}

func TestAnalyzeSupportsOpenAPI30(t *testing.T) {
	for _, version := range []string{"3.0.0", "3.0.3"} {
		t.Run(version, func(t *testing.T) {
			source := []byte(`openapi: ` + version + `
info: {title: Compatible, version: "1"}
paths:
  /items:
    post:
      operationId: createItem
      requestBody:
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Item'}
      responses:
        "200":
          description: item
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Item'}
components:
  schemas:
    Item:
      type: object
      required: [note]
      properties:
        note: {type: string, nullable: true}
`)
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			require.Equal(t, version, document.OpenAPIVersion)
			require.True(t, document.Components.Schemas[0].Schema.Properties[0].Schema.Nullable)
		})
	}
}

func TestAnalyzeSupportsEquivalentRequestMediaTypes(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Flexible, version: "1"}
paths:
  /items:
    post:
      operationId: createItem
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Item'}
          application/x-www-form-urlencoded:
            schema: {$ref: '#/components/schemas/Item'}
          multipart/form-data:
            schema: {$ref: '#/components/schemas/Item'}
      responses: {"204": {description: created}}
components:
  schemas:
    Item:
      type: object
      required: [name]
      properties:
        name: {type: string}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Equal(t, []string{
		"application/json",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
	}, document.Operations[0].RequestBody.ContentTypes)
}

func TestAnalyzeRejectsIncompatibleRequestMediaTypes(t *testing.T) {
	tests := []struct {
		name      string
		media     string
		wantCodes []string
	}{
		{
			name: "different schema",
			media: `
          application/json:
            schema: {type: string}
          multipart/form-data:
            schema: {type: object, properties: {value: {type: string}}}`,
			wantCodes: []string{"request-media-schema"},
		},
		{
			name: "different examples",
			media: `
          application/json:
            example: first
            schema: {type: string}
          application/x-www-form-urlencoded:
            example: second
            schema: {type: string}`,
			wantCodes: []string{"request-media-examples"},
		},
		{
			name: "multipart encoding",
			media: `
          application/json:
            schema: {type: object, properties: {value: {type: string}}}
          multipart/form-data:
            schema: {type: object, properties: {value: {type: string}}}
            encoding:
              value: {contentType: text/plain}`,
			wantCodes: []string{"media-encoding"},
		},
		{
			name: "unsupported media type",
			media: `
          application/json:
            schema: {type: string}
          text/plain:
            schema: {type: string}`,
			wantCodes: []string{"media-type"},
		},
		{
			name: "form media require an object schema",
			media: `
          application/json:
            schema: {type: string}
          application/x-www-form-urlencoded:
            schema: {type: string}`,
			wantCodes: []string{"render-plan"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := []byte(`openapi: 3.1.1
info: {title: Strict, version: "1"}
paths:
  /items:
    post:
      requestBody:
        content:` + test.media + `
      responses: {"204": {description: done}}
`)
			_, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			for _, code := range test.wantCodes {
				requireDiagnosticCode(t, diagnostics, code)
			}
		})
	}
}

func TestAnalyzeSupportsNullableTypeUnions(t *testing.T) {
	tests := []struct {
		name       string
		schemaType string
		format     string
	}{
		{name: "string", schemaType: "string"},
		{name: "uuid", schemaType: "string", format: "uuid"},
		{name: "date-time", schemaType: "string", format: "date-time"},
		{name: "integer", schemaType: "integer"},
		{name: "number", schemaType: "number"},
		{name: "boolean", schemaType: "boolean"},
		{name: "array", schemaType: "array"},
		{name: "object", schemaType: "object"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			format := ""
			if test.format != "" {
				format = "\n          format: " + test.format
			}
			detail := ""
			switch test.schemaType {
			case "array":
				detail = "\n          items: {type: string}"
			case "object":
				detail = "\n          properties: {name: {type: string}}"
			}
			source := []byte(`openapi: 3.1.1
info: {title: Nullable, version: "1"}
paths: {}
components:
  schemas:
    Record:
      type: object
      required: [value]
      properties:
        value:
          type: [` + test.schemaType + `, "null"]` + format + detail + `
`)
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			property := document.Components.Schemas[0].Schema.Properties[0].Schema
			require.Equal(t, test.schemaType, property.Type)
			require.True(t, property.Nullable)
		})
	}
}

func TestAnalyzeRejectsNonNullableTypeUnions(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Union, version: "1"}
paths: {}
components:
  schemas:
    Value: {type: [string, integer]}
`)
	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "schema-type-union")
}

func TestAnalyzeMergesPathParametersWithOperationOverrides(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Overrides, version: "1"}
paths:
  /pets/{id}:
    parameters:
      - name: id
        in: path
        required: true
        description: inherited
        schema: {type: string}
    get:
      parameters:
        - name: id
          in: path
          required: true
          description: operation
          schema: {type: integer, format: int64}
      responses: {"204": {description: done}}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Len(t, document.Operations, 1)
	require.Len(t, document.Operations[0].Parameters, 1)
	parameter := document.Operations[0].Parameters[0]
	require.Equal(t, "operation", parameter.Description)
	require.Equal(t, "integer", parameter.Schema.Type)
}

func TestAnalyzeReportsPlannerDiagnostics(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Strict, version: "1"}
paths:
  /pets/{pet-id}:
    get:
      parameters:
        - name: pet-id
          in: path
          required: false
          deprecated: true
          schema: {type: string}
        - name: trace:id
          in: header
          allowEmptyValue: true
          schema: {type: string}
      responses:
        "200": {description: first}
        "201": {description: second}
`)

	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "parameter-deprecated")
	requireDiagnosticCode(t, diagnostics, "path-parameter-required")
	requireDiagnosticCode(t, diagnostics, "path-parameter-name")
	requireDiagnosticCode(t, diagnostics, "wire-name")
	requireDiagnosticCode(t, diagnostics, "parameter-allow-empty-value")
	requireDiagnosticCode(t, diagnostics, "success-response-count")
}

func TestAnalyzeClassifiesMediaExamplesSeparately(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Examples, version: "1"}
paths:
  /pets:
    get:
      responses:
        "200":
          description: found
          content:
            application/json:
              example: {name: Fido}
              schema: {type: string}
`)

	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Len(t, diagnostics, 1)
	require.Equal(t, "examples", diagnostics[0].Code)
	require.NotEqual(t, "media-metadata", diagnostics[0].Code)
}

func TestAnalyzeRejectsDuplicateAndUnrenderableReferences(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: References, version: "1"}
paths:
  /pets:
    get:
      parameters:
        - name: limit
          in: query
          schema: {type: integer, format: int32}
        - name: limit
          in: query
          schema: {type: integer, format: int32}
        - $ref: "#/components/parameters/Nested"
        - $ref: "#/components/headers/Trace"
      responses:
        "200": {$ref: "#/components/responses/OK"}
        "404":
          description: missing
          headers:
            X-Trace: {$ref: "#/components/headers/Trace"}
    post:
      requestBody: {$ref: "#/components/requestBodies/Body"}
      responses: {"201": {description: created}}
components:
  parameters:
    Nested: {$ref: "#/components/parameters/Limit"}
    Limit:
      name: inheritedLimit
      in: query
      schema: {type: integer, format: int32}
  requestBodies:
    Body:
      content: {application/json: {schema: {type: string}}}
  responses:
    OK: {description: found}
  headers:
    Trace: {schema: {type: string}}
`)

	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "duplicate-parameter")
	requireDiagnosticCode(t, diagnostics, "parameter-reference")
	requireDiagnosticCode(t, diagnostics, "request-body-reference")
	requireDiagnosticCode(t, diagnostics, "response-reference")
	requireDiagnosticCode(t, diagnostics, "header-reference")
}

func TestAnalyzeRejectsUnrenderableComponentKinds(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Component kinds, version: "1"}
paths:
  /pets:
    get:
      responses: {"204": {description: done}}
components:
  requestBodies:
    Body:
      content: {application/json: {schema: {type: string}}}
  responses:
    OK: {description: found}
  headers:
    Trace: {schema: {type: string}}
`)

	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "component-request-body")
	requireDiagnosticCode(t, diagnostics, "component-response")
	requireDiagnosticCode(t, diagnostics, "component-header")
}

func TestAnalyzeRejectsUnrenderableEscapedComponentParameterNames(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Escaped components, version: "1"}
paths:
  /pets:
    get:
      parameters:
        - $ref: "#/components/parameters/Pet~0ID"
        - $ref: "#/components/parameters/Pet~1ID"
      responses: {"204": {description: done}}
components:
  parameters:
    Pet~ID:
      name: tilde
      in: query
      schema: {type: string}
    Pet/ID:
      name: slash
      in: query
      schema: {type: string}
`)

	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "component-parameter-name")
}

func TestLocalComponentReferenceNameDecodesJSONPointerEscapes(t *testing.T) {
	tests := map[string]string{
		"tilde": "Pet~0ID",
		"slash": "Pet~1ID",
	}
	for name, segment := range tests {
		t.Run(name, func(t *testing.T) {
			component, err := localComponentReferenceName("#/components/parameters/"+segment, "#/components/parameters/")
			require.NoError(t, err)
			if name == "tilde" {
				require.Equal(t, "Pet~ID", component)
				return
			}
			require.Equal(t, "Pet/ID", component)
		})
	}
}

func TestAnalyzeRejectsExternalReferencesWithoutWriting(t *testing.T) {
	temp := t.TempDir()
	t.Chdir(temp)
	source := []byte(`openapi: 3.1.1
info: {title: External, version: "1"}
paths:
  /pets:
    get:
      responses:
        "200":
          description: pets
          content:
            application/json:
              schema: {$ref: ./models.yaml#/Pet}
components:
  schemas:
    Owner: {$ref: https://example.com/models.yaml#/Owner}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Nil(t, document)
	require.Len(t, diagnostics, 2)
	requireDiagnosticCode(t, diagnostics, "external-reference")
	entries, err := os.ReadDir(temp)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestAnalyzeReportsMalformedInput(t *testing.T) {
	document, diagnostics, err := Analyze([]byte("openapi: ["))
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrUnsupportedVersion))
	require.Nil(t, document)
	require.Empty(t, diagnostics)
}

func TestAnalyzeAcceptsSchemaDefaultAndDeprecated(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Defaults, version: "1"}
paths:
  /pets:
    get:
      responses: {"204": {description: done}}
components:
  schemas:
    FaceCoordinatesDto:
      type: object
      properties:
        mouthX:
          type: number
          format: float
          deprecated: true
        stable:
          type: string
          readOnly: true
        input:
          type: string
          writeOnly: true
        apiVersion:
          type: string
          default: "1.0"
        retries:
          type: integer
          format: int32
          default: 3
        enabled:
          type: boolean
          default: true
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)

	byName := map[string]*Schema{}
	for _, property := range document.Components.Schemas[0].Schema.Properties {
		byName[property.Name] = property.Schema
	}
	require.True(t, byName["mouthX"].Deprecated)
	require.True(t, byName["stable"].ReadOnly)
	require.True(t, byName["input"].WriteOnly)
	require.NotNil(t, byName["apiVersion"].Default)
	require.Equal(t, "1.0", byName["apiVersion"].Default.Value)
	require.NotNil(t, byName["retries"].Default)
	require.Equal(t, 3, byName["retries"].Default.Value)
	require.NotNil(t, byName["enabled"].Default)
	require.Equal(t, true, byName["enabled"].Default.Value)
}

func TestAnalyzeRejectsDefaultForCompositeTypesAndMismatchedValues(t *testing.T) {
	tests := map[string]string{
		"object default": `
    Widget:
      type: object
      default: {}
      properties:
        name: {type: string}
`,
		"array default": `
    Widget:
      type: array
      default: []
      items: {type: string}
`,
		"mismatched scalar default": `
    Widget:
      type: integer
      format: int32
      default: "not-a-number"
`,
	}
	for name, schema := range tests {
		t.Run(name, func(t *testing.T) {
			source := []byte(fmt.Sprintf(`openapi: 3.1.1
info: {title: Defaults, version: "1"}
paths:
  /pets:
    get:
      responses: {"204": {description: done}}
components:
  schemas:%s`, schema))

			_, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			requireDiagnosticCode(t, diagnostics, "schema-keyword")
		})
	}
}

func TestAnalyzeClassifiesParameterAndHeaderDeprecatedAsLossy(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Deprecated, version: "1"}
paths:
  /pets:
    get:
      parameters:
        - name: api-version
          in: header
          required: true
          deprecated: true
          schema: {type: string}
      responses:
        "200":
          description: found
          headers:
            X-Legacy:
              deprecated: true
              schema: {type: string}
          content:
            application/json:
              schema: {type: string}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.NotNil(t, document)
	requireDiagnosticCode(t, diagnostics, "parameter-deprecated")
	requireDiagnosticCode(t, diagnostics, "header-deprecated")

	fatal, warnings := diagnostics.Classify(true)
	require.Empty(t, fatal)
	require.Len(t, warnings, 2)
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return source
}

func requireDiagnosticCode(t *testing.T, diagnostics Diagnostics, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Errorf("diagnostic code %q not found in %#v", code, diagnostics)
}

func TestAnalyzeDoesNotRetainParserSpecificTypes(t *testing.T) {
	document, diagnostics, err := Analyze(readFixture(t, "supported.yaml"))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.NotContains(t, reflect.TypeOf(*document).PkgPath(), "libopenapi")
}
