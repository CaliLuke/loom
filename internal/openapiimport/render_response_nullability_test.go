package openapiimport

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

// responseNullabilityPaths cover each success response body occurrence rendered
// by responseSchemaBody: Body and OpenAPIBody for nullable scalar references,
// object references, inline objects and raw media, with nonnullable and
// nullable-declaration controls.
const responseNullabilityPaths = `openapi: 3.1.1
info: {title: Nullable, version: "1"}
paths:
  /scalar:
    get:
      operationId: scalar
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {anyOf: [{$ref: '#/components/schemas/Short'}, {type: 'null'}]}
              examples: {empty: {value: null}, fit: {value: xy}}
  /scalar-bare:
    get:
      operationId: scalarBare
      responses:
        "200": {description: ok, content: {application/json: {schema: {anyOf: [{$ref: '#/components/schemas/Short'}, {type: 'null'}]}}}}
  /object:
    get:
      operationId: object
      responses:
        "200": {description: ok, content: {application/json: {schema: {anyOf: [{$ref: '#/components/schemas/Obj'}, {type: 'null'}]}, example: null}}}
  /inline:
    get:
      operationId: inline
      responses:
        "200": {description: ok, content: {application/json: {schema: {anyOf: [{type: object, properties: {name: {type: string}}}, {type: 'null'}]}, example: null}}}
  /raw:
    get:
      operationId: raw
      responses:
        "200": {description: ok, content: {text/plain: {schema: {anyOf: [{$ref: '#/components/schemas/Short'}, {type: 'null'}]}}}}
  /plain:
    get:
      operationId: plain
      responses:
        "200": {description: ok, content: {application/json: {schema: {$ref: '#/components/schemas/Short'}, example: xy}}}
  /named:
    get:
      operationId: named
      responses:
        "200": {description: ok, content: {application/json: {schema: {$ref: '#/components/schemas/NullableShort'}, example: null}}}
`

// responseNullabilityFailurePath covers the same occurrences for error
// responses. Generated error codecs do not accept nullable error roots, so
// only evaluation covers it.
const responseNullabilityFailurePath = `  /failure:
    get:
      operationId: failure
      responses:
        "204": {description: done}
        "400": {description: bad, content: {application/json: {schema: {anyOf: [{$ref: '#/components/schemas/Short'}, {type: 'null'}]}, example: null}}}
        "409": {description: conflict, content: {application/json: {schema: {anyOf: [{$ref: '#/components/schemas/Obj'}, {type: 'null'}]}, example: null}}}
`

const responseNullabilityComponents = `components:
  schemas:
    Short: {type: string, minLength: 2}
    Obj: {type: object, properties: {name: {type: string}}}
    NullableShort: {anyOf: [{type: string, minLength: 2}, {type: 'null'}]}
`

// responseNullabilityChecks asserts the evaluated HTTP body occurrences carry
// the response nullability while the referenced declarations keep their own.
const responseNullabilityChecks = `
service := expr.Root.API.HTTP.Services[0]
body := func(method string, status int, openAPI bool) *expr.AttributeExpr {
 endpoint := service.Endpoint(method)
 if endpoint == nil {
  t.Fatalf("missing endpoint %s", method)
 }
 responses := append([]*expr.HTTPResponseExpr(nil), endpoint.Responses...)
 for _, failure := range endpoint.HTTPErrors {
  responses = append(responses, failure.Response)
 }
 for _, response := range responses {
  if response.StatusCode != status {
   continue
  }
  if openAPI {
   return response.OpenAPIBody
  }
  return response.Body
 }
 t.Fatalf("missing %s %d response", method, status)
 return nil
}
for _, tc := range []struct {
 method   string
 status   int
 openAPI  bool
 nullable bool
}{
 {"Scalar", 200, true, true},
 {"ScalarBare", 200, true, true},
 {"Object", 200, false, true},
 {"Object", 200, true, true},
 {"Inline", 200, false, true},
 {"Failure", 400, true, true},
 {"Failure", 409, false, true},
 {"Failure", 409, true, true},
 {"Raw", 200, true, true},
 {"Plain", 200, true, false},
 {"Named", 200, true, true},
} {
 attribute := body(tc.method, tc.status, tc.openAPI)
 if attribute == nil {
  t.Errorf("%s %d openAPI=%t body is missing", tc.method, tc.status, tc.openAPI)
  continue
 }
 if got := expr.IsNullable(attribute); got != tc.nullable {
  t.Errorf("%s %d openAPI=%t body nullable = %t, want %t", tc.method, tc.status, tc.openAPI, got, tc.nullable)
 }
}
for _, method := range service.ServiceExpr.Methods {
 if method.Name == "Failure" {
  for _, failure := range method.Errors {
   if !expr.IsNullable(failure.AttributeExpr) {
    t.Errorf("error %s lost nullability", failure.Name)
   }
  }
  continue
 }
 if got, want := expr.IsNullable(method.Result), method.Name != "Plain" && method.Name != "Raw"; got != want {
  t.Errorf("%s result nullable = %t, want %t", method.Name, got, want)
 }
}
for name, nullable := range map[string]bool{"Short": false, "Obj": false, "NullableShort": true} {
 declaration := expr.Root.UserType(name)
 if declaration == nil {
  t.Fatalf("missing declaration %s", name)
 }
 if got := expr.IsNullable(declaration.Attribute()); got != nullable {
  t.Errorf("declaration %s nullable = %t, want %t", name, got, nullable)
 }
}
short := expr.Root.UserType("Short").Attribute()
if short.Validation == nil || short.Validation.MinLength == nil || *short.Validation.MinLength != 2 {
 t.Error("declaration Short lost its validation")
}
`

func TestRenderResponseBodyNullability(t *testing.T) {
	render := func(t *testing.T, source string) []byte {
		t.Helper()
		document, diagnostics, err := Analyze([]byte(source))
		require.NoError(t, err)
		require.Empty(t, diagnostics)
		rendered, err := Render(document, Options{PackageName: "design"})
		require.NoError(t, err)
		return rendered
	}
	t.Run("evaluate", func(t *testing.T) {
		rendered := render(t, responseNullabilityPaths+responseNullabilityFailurePath+responseNullabilityComponents)
		requireRenderedDesignEvaluates(t, rendered, 8, responseNullabilityChecks)
	})
	t.Run("generate", func(t *testing.T) {
		moduleDir := requireRenderedDesignGenerates(t, render(t, responseNullabilityPaths+responseNullabilityComponents))
		for _, name := range []string{"openapi.json", "openapi.yaml"} {
			content, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", name))
			require.NoError(t, err)
			parsed, err := libopenapi.NewDocument(content)
			require.NoError(t, err)
			_, err = parsed.BuildV3Model()
			require.NoError(t, err)
		}
		contract := readGeneratedOpenAPIContract(t, moduleDir)
		for _, tc := range []struct {
			path, status, media string
			nullable            bool
		}{
			{"/scalar", "200", "application/json", true},
			{"/scalar-bare", "200", "application/json", true},
			{"/object", "200", "application/json", true},
			{"/inline", "200", "application/json", true},
			{"/raw", "200", "text/plain", true},
			{"/plain", "200", "application/json", false},
			{"/named", "200", "application/json", true},
		} {
			response := contract["paths"].(map[string]any)[tc.path].(map[string]any)["get"].(map[string]any)["responses"].(map[string]any)[tc.status].(map[string]any)
			schema := response["content"].(map[string]any)[tc.media].(map[string]any)["schema"].(map[string]any)
			require.Equal(t, tc.nullable, contractSchemaAdmitsNull(t, contract, schema), "%s %s schema %v", tc.path, tc.status, schema)
		}
		for name, nullable := range map[string]bool{"Short": false, "Obj": false, "NullableShort": true} {
			schema := referencedSchema(t, contract, "#/components/schemas/"+name)
			require.Equal(t, nullable, contractSchemaAdmitsNull(t, contract, schema), "component %s schema %v", name, schema)
		}
	})
}

// contractSchemaAdmitsNull reports whether a generated OpenAPI schema admits
// JSON null through its type, a null alternative, a referenced component or
// an allOf whose every member admits null.
func contractSchemaAdmitsNull(t *testing.T, contract, schema map[string]any) bool {
	t.Helper()
	if ref, ok := schema["$ref"].(string); ok {
		return contractSchemaAdmitsNull(t, contract, referencedSchema(t, contract, ref))
	}
	switch kind := schema["type"].(type) {
	case string:
		if kind == "null" {
			return true
		}
	case []any:
		for _, item := range kind {
			if item == "null" {
				return true
			}
		}
	}
	for _, key := range []string{"anyOf", "oneOf"} {
		branches, _ := schema[key].([]any)
		for _, branch := range branches {
			if contractSchemaAdmitsNull(t, contract, branch.(map[string]any)) {
				return true
			}
		}
	}
	branches, _ := schema["allOf"].([]any)
	for _, branch := range branches {
		if !contractSchemaAdmitsNull(t, contract, branch.(map[string]any)) {
			return false
		}
	}
	return len(branches) > 0
}
