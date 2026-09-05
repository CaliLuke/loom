package openapiimport

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderSupportedFixtureDeterministically(t *testing.T) {
	yamlDocument, yamlDiagnostics, err := Analyze(readFixture(t, "supported.yaml"))
	require.NoError(t, err)
	require.Empty(t, yamlDiagnostics)
	jsonDocument, jsonDiagnostics, err := Analyze(readFixture(t, "supported.json"))
	require.NoError(t, err)
	require.Empty(t, jsonDiagnostics)

	yamlSource, err := Render(yamlDocument, Options{PackageName: "design"})
	require.NoError(t, err)
	jsonSource, err := Render(jsonDocument, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Equal(t, yamlSource, jsonSource)

	parsed, err := parser.ParseFile(token.NewFileSet(), "design.go", yamlSource, parser.AllErrors)
	require.NoError(t, err)
	require.Equal(t, "design", parsed.Name.Name)
	for _, want := range []string{
		`var ImportedNewPet = Type("NewPet", func() {`,
		`Meta("openapi:additionalProperties", "false")`,
		`Meta("openapi:example", "false")`,
		`Meta("openapi:tag:pets")`,
		`Method("LegacyGet", func() {`,
		`GET("/legacy")`,
		`Response(302, func() {`,
		`Method("PetsCreate", func() {`,
		`Meta("openapi:operationId", "pets.create")`,
		`Meta("openapi:summary", "")`,
		`Meta("openapi:description:requestBody", "Pet to create.")`,
		`POST("/pets")`,
		`Response(201, func() {`,
		`Method("PetsGet", func() {`,
		`Error("Status404", ImportedProblem)`,
		`GET("/pets/{pet_id}")`,
		`Param("pet_id")`,
		`Header("xTraceID:X-Trace-ID")`,
		`Meta("openapi:component:parameter", "PetID")`,
		`Meta("openapi:allowEmptyValue", "false")`,
		`Response("Status404", 404, func() {`,
		`Meta("openapi:description:errorName", "false")`,
		`Meta("openapi:readOnly", "true")`,
		`Meta("openapi:writeOnly", "true")`,
		`Meta("openapi:deprecated", "true")`,
		`Meta("openapi:typename:canonical", "true")`,
		`Default("cat")`,
		`Attribute("weight", Float64, func() {`,
		`Attribute("stock", Int, func() {`,
		`var ImportedStatus = Type("Status", Int, func() {`,
		`Attribute("status", ImportedStatus)`,
		`Meta("openapi:format", "")`,
	} {
		require.Contains(t, string(yamlSource), want)
	}
	require.Contains(t, string(yamlSource), "HTTP(func() {\n\t\t\tMeta(\"openapi:tag:pets\")")
	require.Contains(t, string(yamlSource), "Attribute(\"body\", ImportedNewPet, func() {\n\t\t\t\tMeta(\"openapi:requestBody:extension:x-request-note\"")
	require.Contains(t, string(yamlSource), "\t\t\t\tMeta(\"openapi:description:requestBody\", \"Pet to create.\")\n\t\t\t})")
	require.NotContains(t, string(yamlSource), "Payload(func() {\n\t\t\tMeta(\"openapi:description:requestBody\"")
}

func TestRenderURIReferenceFormat(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Downloads, version: "1"}
paths:
  /download:
    get:
      operationId: download
      responses:
        "200":
          description: Download location.
          content:
            application/json:
              schema:
                type: string
                format: uri-reference
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(rendered), "Format(FormatURIReference)")
	requireRenderedDesignEvaluates(t, rendered, 1)
}

func TestRenderSuppressesGeneratedOperationMetadataWhenAbsent(t *testing.T) {
	document, diagnostics, err := Analyze(readFixture(t, "supported.yaml"))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	document.Operations[1].OperationID = ""
	document.Operations[1].Summary = ""

	source, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	method := string(source)
	start := strings.Index(method, `Method("PetsCreate", func() {`)
	require.NotEqual(t, -1, start)
	method = method[start:]
	end := strings.Index(method, `Method("PetsGet", func() {`)
	require.NotEqual(t, -1, end)
	method = method[:end]
	require.Contains(t, method, `Meta("openapi:operationId", "")`)
	require.Contains(t, method, `Meta("openapi:summary", "")`)
}

func TestRenderOutputCompilesAndEvaluatesAsLoomDesign(t *testing.T) {
	document, diagnostics, err := Analyze(readFixture(t, "supported.yaml"))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	source, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, source, 3)
}

func TestRenderDisambiguatesSharedErrorStatusesByOperation(t *testing.T) {
	document, diagnostics, err := Analyze(readFixture(t, "shared_status_different_schemas.yaml"))
	require.NoError(t, err)
	require.Empty(t, diagnostics)

	source, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	design := string(source)
	for _, want := range []string{
		`Error("ListItemsStatus400", ImportedInvalidCursor)`,
		`Response("ListItemsStatus400", 400, func() {`,
		`Error("CreateItemStatus400", MapOf(String, ArrayOf(String)))`,
		`Response("CreateItemStatus400", 400, func() {`,
	} {
		require.Contains(t, design, want)
	}
	require.NotContains(t, design, `Error("Status400"`)
	requireRenderedDesignEvaluates(t, source, 2)
}

func TestRenderRoundTripsAPIKeySecurityContracts(t *testing.T) {
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
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	design := string(rendered)
	require.Contains(t, design, `APIKeySecurity("HeaderKey"`)
	require.Contains(t, design, `APIKeySecurity("CookieKey"`)
	require.Contains(t, design, `Security(ImportedHeaderKeySecurity)`)
	require.Contains(t, design, `NoSecurity()`)
	require.Contains(t, design, "Security(ImportedCookieKeySecurity)\n\t\tSecurity()")
	require.Contains(t, design, `Header("headerKeyCredential:X-API-Key")`)
	require.Contains(t, design, `Cookie("cookieKeyCredential:session-id")`)
	requireRenderedDesignEvaluates(t, rendered, 3)

	moduleDir := requireRenderedDesignGenerates(t, rendered)
	generated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	var contract map[string]any
	require.NoError(t, json.Unmarshal(generated, &contract))
	securitySchemes := contract["components"].(map[string]any)["securitySchemes"].(map[string]any)
	require.Equal(t, "header", securitySchemes["HeaderKey"].(map[string]any)["in"])
	require.Equal(t, "cookie", securitySchemes["CookieKey"].(map[string]any)["in"])
	require.Equal(t, []any{map[string]any{"HeaderKey": []any{}}}, contract["security"])
	require.Equal(t, []any{}, operationFromImportedSpec(t, contract, "/public", "get")["security"])
	require.Equal(t, []any{
		map[string]any{"CookieKey": []any{}},
		map[string]any{},
	}, operationFromImportedSpec(t, contract, "/optional", "get")["security"])
}

func TestRenderPreservesSchemaAndMediaExamples(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Examples, version: "1"}
paths:
  /pets:
    post:
      operationId: createPet
      requestBody:
        required: true
        content:
          application/json:
            example: {name: Fido}
            schema: {$ref: '#/components/schemas/Pet'}
      responses:
        "200":
          description: created
          content:
            application/json:
              examples:
                created:
                  summary: Created pet
                  description: A newly-created pet.
                  value: {name: Fido}
              schema: {$ref: '#/components/schemas/Pet'}
components:
  schemas:
    Pet:
      type: object
      required: [name]
      properties:
        name: {type: string, example: Fido}
        kind: {type: string, examples: [dog, cat]}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, rendered, 1)

	design := string(rendered)
	for _, expected := range []string{
		`Example("Fido")`,
		`Example("example-1", "dog")`,
		`Example("example-2", "cat")`,
		`Example(Val{"name": "Fido"})`,
		`Example("created", func() {`,
		`Meta("openapi:example:summary", "Created pet")`,
		`Description("A newly-created pet.")`,
		`Value(Val{"name": "Fido"})`,
	} {
		require.Contains(t, design, expected)
	}
}

func TestRenderPreservesSchemaTitles(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Titles, version: "1"}
paths:
  /items:
    post:
      operationId: createItem
      requestBody:
        required: true
        content:
          application/json:
            schema:
              title: Create Item Request
              type: string
      responses:
        "200":
          description: created
          content:
            application/json:
              schema:
                title: Create Item Response
                type: object
                properties:
                  item: {$ref: '#/components/schemas/Item'}
components:
  schemas:
    Item:
      title: Item Resource
      type: object
      required: [updated_at]
      properties:
        updated_at:
          title: Last Modified At
          type: string
        tags:
          type: array
          items:
            title: Item Tag
            type: string
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)

	design := string(rendered)
	for _, title := range []string{
		"Item Resource",
		"Last Modified At",
		"Create Item Request",
		"Item Tag",
		"Create Item Response",
	} {
		require.Contains(t, design, `Title("`+title+`")`)
	}
	requireRenderedDesignEvaluates(t, rendered, 1)
	moduleDir := requireRenderedDesignGenerates(t, rendered)

	generated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	var contract map[string]any
	require.NoError(t, json.Unmarshal(generated, &contract))

	item := contract["components"].(map[string]any)["schemas"].(map[string]any)["Item"].(map[string]any)
	require.Equal(t, "Item Resource", item["title"])
	require.Equal(t, "Last Modified At", item["properties"].(map[string]any)["updated_at"].(map[string]any)["title"])

	operation := operationFromImportedSpec(t, contract, "/items", "post")
	requestSchema := operation["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	require.Equal(t, "Create Item Request", requestSchema["title"])
	require.Equal(t, "Item Tag", item["properties"].(map[string]any)["tags"].(map[string]any)["items"].(map[string]any)["title"])
	responseSchema := operation["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	responseTitle := responseSchema["title"]
	if ref, ok := responseSchema["$ref"].(string); ok {
		name := strings.TrimPrefix(ref, "#/components/schemas/")
		require.NotEqual(t, ref, name)
		responseTitle = contract["components"].(map[string]any)["schemas"].(map[string]any)[name].(map[string]any)["title"]
	}
	require.Equal(t, "Create Item Response", responseTitle)
}

func requireRenderedDesignGenerates(t *testing.T, source []byte) string {
	t.Helper()
	moduleDir := t.TempDir()
	repoRoot := repositoryRoot(t)
	goMod := "module example.com/imported\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\n" +
		"replace github.com/CaliLuke/loom => " + repoRoot + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(goMod), 0o600))
	designDir := filepath.Join(moduleDir, "design")
	require.NoError(t, os.MkdirAll(designDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(designDir, "design.go"), source, 0o600))

	loomBin := filepath.Join(moduleDir, "loom")
	build := exec.Command("go", "build", "-o", loomBin, "./cmd/loom")
	build.Dir = repoRoot
	output, err := build.CombinedOutput()
	require.NoError(t, err, string(output))

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = moduleDir
	output, err = tidy.CombinedOutput()
	require.NoError(t, err, string(output))

	generate := exec.Command(loomBin, "gen", "example.com/imported/design", "-o", ".")
	generate.Dir = moduleDir
	output, err = generate.CombinedOutput()
	require.NoError(t, err, "%s\n%s", output, source)

	test := exec.Command("go", "test", "-mod=mod", "./...")
	test.Dir = moduleDir
	output, err = test.CombinedOutput()
	require.NoError(t, err, string(output))
	return moduleDir
}

func TestRenderRoundTripsMultipleRequestMediaTypes(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Flexible, version: "1"}
paths:
  /items:
    post:
      operationId: createItem
      requestBody:
        required: true
        description: One schema with three encodings.
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
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	design := string(rendered)
	require.Contains(t, design, "SkipRequestBodyEncodeDecode()")
	require.Contains(t, design, `OpenAPIRequestBodyTypes(ImportedItem, []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"}, true`)
	require.NotContains(t, design, `Body("body")`)
	requireRenderedDesignEvaluates(t, rendered, 1)
	moduleDir := requireRenderedDesignGenerates(t, rendered)

	generated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	var contract map[string]any
	require.NoError(t, json.Unmarshal(generated, &contract))
	paths := contract["paths"].(map[string]any)
	operation := paths["/items"].(map[string]any)["post"].(map[string]any)
	body := operation["requestBody"].(map[string]any)
	content := body["content"].(map[string]any)
	contentTypes := make([]string, 0, len(content))
	for contentType := range content {
		contentTypes = append(contentTypes, contentType)
	}
	require.ElementsMatch(t, []string{
		"application/json",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
	}, contentTypes)
}

func TestRenderMultipleRequestMediaTypesWithInlineObject(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Inline, version: "1"}
paths:
  /items:
    post:
      requestBody:
        content:
          application/json:
            schema: &item
              type: object
              required: [name]
              properties:
                name: {type: string}
          multipart/form-data:
            schema: *item
      responses: {"204": {description: created}}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(rendered), "OpenAPIRequestBodyTypes(func() {")
	requireRenderedDesignEvaluates(t, rendered, 1)
}

func TestRenderPreservesVendorExtensionsAtSupportedScopes(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Extensions, version: "1"}
x-document: {reviewed: true}
paths:
  /pets:
    post:
      operationId: createPet
      x-operation: [one, two]
      parameters:
        - name: trace
          in: header
          required: false
          x-parameter: 7
          schema:
            type: string
            x-parameter-schema: {kind: trace}
      requestBody:
        required: true
        x-request: request-value
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Pet'}
      responses:
        "201":
          description: created
          x-response: {state: created}
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Pet'}
components:
  schemas:
    Pet:
      type: object
      x-schema: {audience: public}
      required: [name]
      properties:
        name: {type: string}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Equal(t, map[string]any{"reviewed": true}, document.Extensions["x-document"])
	require.Equal(t, []any{"one", "two"}, document.Operations[0].Extensions["x-operation"])
	require.Equal(t, 7, document.Operations[0].Parameters[0].Extensions["x-parameter"])
	require.Equal(t, "request-value", document.Operations[0].RequestBody.Extensions["x-request"])
	require.Equal(t, map[string]any{"state": "created"}, document.Operations[0].Responses[0].Response.Extensions["x-response"])
	require.Equal(t, map[string]any{"audience": "public"}, document.Components.Schemas[0].Schema.Extensions["x-schema"])

	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, rendered, 1)
	design := string(rendered)
	for _, expected := range []string{
		`Meta("openapi:document:extension:x-document", "{\"reviewed\":true}")`,
		`Meta("openapi:extension:x-operation", "[\"one\",\"two\"]")`,
		`Meta("openapi:parameter:extension:x-parameter", "7")`,
		`Meta("openapi:schema:extension:x-parameter-schema", "{\"kind\":\"trace\"}")`,
		`Meta("openapi:requestBody:extension:x-request", "\"request-value\"")`,
		`Meta("openapi:response:extension:x-response", "{\"state\":\"created\"}")`,
		`Meta("openapi:schema:extension:x-schema", "{\"audience\":\"public\"}")`,
	} {
		require.Contains(t, design, expected)
	}
}

func TestRenderPreservesNullableSchemas(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Nullable, version: "1"}
paths:
  /records:
    post:
      operationId: createRecord
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Record'}
      responses:
        "200":
          description: record
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Record'}
components:
  schemas:
    Record:
      type: object
      required: [required_note, required_child]
      examples:
        - required_note: null
          required_child: null
          nullable_items: [value, null]
          nullable_values: {value: null}
          nullable_objects: [{name: loom}, null]
          nullable_object_values: {value: null}
      properties:
        required_note:
          type: [string, "null"]
          examples: [null]
        optional_count: {type: [integer, "null"]}
        required_child:
          type: [object, "null"]
          properties:
            name: {type: string}
        optional_tags:
          type: [array, "null"]
          items: {type: string}
        nullable_items:
          type: array
          items: {type: [string, "null"]}
        nullable_values:
          type: object
          additionalProperties: {type: [integer, "null"]}
        nullable_objects:
          type: array
          items:
            anyOf:
              - {$ref: '#/components/schemas/Child'}
              - {type: "null"}
        nullable_object_values:
          type: object
          additionalProperties:
            anyOf:
              - {$ref: '#/components/schemas/Child'}
              - {type: "null"}
        optional_label: {$ref: '#/components/schemas/NullableLabel'}
    Child:
      type: object
      properties:
        name: {type: string}
    NullableLabel:
      type: [string, "null"]
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, rendered, 1)
	requireRenderedDesignGenerates(t, rendered)
	design := string(rendered)
	require.GreaterOrEqual(t, strings.Count(design, "Nullable()"), 5)
	require.NotContains(t, design, `Meta("openapi:nullable"`)
	require.NotContains(t, design, `Meta("struct:field:type", "loom.Nullable[`)
	require.Contains(t, design, "ArrayOf(String, func() {\n\t\tNullable()")
	require.Contains(t, design, "MapOf(String, Int, func() {\n\t\tElem(func() {\n\t\t\tNullable()")
	require.Contains(t, design, "Null())")
	require.Contains(t, design, "Attribute(\"optional_label\", ImportedNullableLabel, func() {\n\t\tNullable()")
	require.Contains(t, design, `Example("example-1", Val{"nullable_items": []any{"value", nil}, "nullable_object_values": Val{"value": nil}, "nullable_objects": []any{Val{"name": "loom"}, nil}, "nullable_values": Val{"value": nil}, "required_child": nil, "required_note": nil})`)
	require.Contains(t, design, "var ImportedNullableLabel = Type(\"NullableLabel\", String, func() {\n\tMeta(\"openapi:typename:canonical\", \"true\")\n\tNullable()")
}

func TestExampleLiteralKeepsNestedNullAsGoNil(t *testing.T) {
	literal, err := exampleLiteral(map[string]any{
		"items": []any{"value", nil},
		"value": nil,
	})
	require.NoError(t, err)
	require.Equal(t, `Val{"items": []any{"value", nil}, "value": nil}`, literal)

	literal, err = exampleLiteral(nil)
	require.NoError(t, err)
	require.Equal(t, "Null()", literal)
}

func requireRenderedDesignEvaluates(t *testing.T, source []byte, wantMethods int) {
	t.Helper()
	moduleDir := t.TempDir()
	repoRoot := repositoryRoot(t)
	goMod := "module example.com/imported\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\n" +
		"replace github.com/CaliLuke/loom => " + repoRoot + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(goMod), 0o600))
	designDir := filepath.Join(moduleDir, "design")
	require.NoError(t, os.MkdirAll(designDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(designDir, "design.go"), source, 0o600))
	testSource := fmt.Sprintf(`package design

import (
	"testing"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestEvaluateImportedDesign(t *testing.T) {
	if err := expr.RegisterDefaultRoots(); err != nil {
		t.Fatal(err)
	}
	if err := eval.RunDSL(); err != nil {
		t.Fatal(err)
	}
	if got := len(expr.Root.Services); got != 1 {
		t.Fatalf("got %%d services, want 1", got)
	}
	if got := len(expr.Root.Services[0].Methods); got != %d {
		t.Fatalf("got %%d methods, want %d", got)
	}
}
`, wantMethods, wantMethods)
	require.NoError(t, os.WriteFile(filepath.Join(designDir, "design_test.go"), []byte(testSource), 0o600))

	cmd := exec.Command("go", "test", "-mod=mod", "./design")
	cmd.Dir = moduleDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

func TestRenderMultipartFormAndBinaryMediaTypes(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Media API, version: "1"}
paths:
  /uploads:
    post:
      operationId: upload
      requestBody:
        required: true
        content:
          multipart/form-data:
            schema:
              $ref: '#/components/schemas/Upload'
      responses: {"204": {description: uploaded}}
  /tokens:
    post:
      operationId: createToken
      requestBody:
        required: true
        content:
          application/x-www-form-urlencoded:
            schema:
              type: object
              required: [grant_type]
              properties:
                grant_type: {type: string}
      responses: {"204": {description: created}}
  /documents/{id}:
    get:
      operationId: download
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      responses:
        "200":
          description: document
          content:
            application/pdf:
              schema: {type: string, format: binary}
  /signatures:
    post:
      operationId: attachSignature
      responses:
        "200":
          description: signed document
          content:
            application/pdf:
              schema: {type: string, format: binary}
        "400":
          description: invalid signature
          content:
            application/pdf:
              schema: {type: string, format: binary}
components:
  schemas:
    Upload:
      type: object
      required: [file, label]
      properties:
        file: {type: string, format: binary}
        label: {type: string}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, rendered, 4)

	design := string(rendered)
	for _, expected := range []string{
		`Attribute("file", Bytes)`,
		`Attribute("label", String)`,
		`Extend(ImportedUpload)`,
		`MultipartRequest()`,
		`Attribute("grant_type", String)`,
		`FormRequest()`,
		`Consumes("application/x-www-form-urlencoded", "multipart/form-data")`,
		`Produces("application/pdf")`,
		`FileResponse()`,
		`SkipResponseBodyEncodeDecode()`,
		`ContentType("application/pdf")`,
		`OpenAPIBody(Bytes)`,
	} {
		require.Contains(t, design, expected)
	}
	require.NotContains(t, design, `Body("body")`)
}

func TestRenderErrorFieldCollisionGeneratesCompilingService(t *testing.T) {
	schemaRef := func(name string) *Schema {
		return &Schema{Ref: "#/components/schemas/" + name}
	}
	document := &Document{
		OpenAPIVersion: "3.1.1",
		Title:          "Error Fields",
		APIVersion:     "1.0.0",
		Components: Components{Schemas: []NamedSchema{
			{
				Name:   "Failure",
				GoName: "Failure",
				Schema: &Schema{Type: "object", Properties: []NamedProperty{
					{Name: "error", Schema: &Schema{Type: "string"}},
					{Name: "loomErrorName", Schema: &Schema{Type: "string"}},
				}},
			},
			{
				Name:   "Success",
				GoName: "Success",
				Schema: &Schema{Type: "object", Properties: []NamedProperty{
					{Name: "error", Schema: &Schema{Type: "string"}},
				}},
			},
		}},
		Operations: []Operation{{
			Method: "GET", Path: "/fields", OperationID: "fields.get", GoName: "FieldsGet",
			Responses: []StatusResponse{
				{Status: "200", Response: Response{ContentType: "application/json", Schema: schemaRef("Success")}},
				{Status: "500", Response: Response{ContentType: "application/json", Schema: schemaRef("Failure")}},
			},
		}, {
			Method: "GET", Path: "/inline", OperationID: "inline.get", GoName: "InlineGet",
			Responses: []StatusResponse{
				{Status: "204", Response: Response{}},
				{
					Status: "400",
					Response: Response{
						ContentType: "application/json",
						Schema: &Schema{Type: "object", Properties: []NamedProperty{
							{Name: "error", Schema: &Schema{Type: "string"}},
						}},
					},
				},
			},
		}},
	}

	source, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(source), "Meta(\"struct:field:name\", \"ErrorField\")")
	require.Contains(t, string(source), "Meta(\"struct:field:name\", \"LoomErrorNameField\")")
	require.Equal(t, 3, strings.Count(string(source), "struct:field:name"))

	moduleDir := t.TempDir()
	repoRoot := repositoryRoot(t)
	goMod := "module example.com/imported\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\n" +
		"replace github.com/CaliLuke/loom => " + repoRoot + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(goMod), 0o600))
	designDir := filepath.Join(moduleDir, "design")
	require.NoError(t, os.MkdirAll(designDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(designDir, "design.go"), source, 0o600))

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = moduleDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	cmd = exec.Command("go", "run", "-mod=mod", "github.com/CaliLuke/loom/cmd/loom", "gen", "example.com/imported/design", "-o", ".")
	cmd.Dir = moduleDir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	cmd = exec.Command("go", "test", "-mod=mod", "./...")
	cmd.Dir = moduleDir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

// TestRenderRoundTripsNewlySupportedSchemaKeywordsIntoOpenAPI regenerates the
// imported design and inspects the generated OpenAPI document, confirming
// that default, deprecated, readOnly, writeOnly, and unformatted
// integer/number properties imported from supported.yaml reappear in the
// regenerated contract rather than merely compiling.
func TestRenderRoundTripsNewlySupportedSchemaKeywordsIntoOpenAPI(t *testing.T) {
	document, diagnostics, err := Analyze(readFixture(t, "supported.yaml"))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	source, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)

	moduleDir := t.TempDir()
	repoRoot := repositoryRoot(t)
	goMod := "module example.com/imported\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\n" +
		"replace github.com/CaliLuke/loom => " + repoRoot + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(goMod), 0o600))
	designDir := filepath.Join(moduleDir, "design")
	require.NoError(t, os.MkdirAll(designDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(designDir, "design.go"), source, 0o600))

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = moduleDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	cmd = exec.Command("go", "run", "-mod=mod", "github.com/CaliLuke/loom/cmd/loom", "gen", "example.com/imported/design", "-o", ".")
	cmd.Dir = moduleDir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	openAPI, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)

	// weight (unformatted number) and stock (unformatted integer) use Loom's
	// widest representations while preserving the source contract's absent
	// format.
	var decoded struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					Type       string `json:"type"`
					Format     string `json:"format"`
					Default    any    `json:"default"`
					Deprecated bool   `json:"deprecated"`
					ReadOnly   bool   `json:"readOnly"`
					WriteOnly  bool   `json:"writeOnly"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	require.NoError(t, json.Unmarshal(openAPI, &decoded))
	newPet, ok := decoded.Components.Schemas["NewPet"]
	require.True(t, ok, "expected the canonical NewPet schema")
	require.Equal(t, "cat", newPet.Properties["kind"].Default)
	pet, ok := decoded.Components.Schemas["Pet"]
	require.True(t, ok, "expected the canonical Pet schema")
	require.True(t, pet.Properties["nickname"].Deprecated)
	require.True(t, pet.Properties["id"].ReadOnly)
	require.True(t, pet.Properties["secret"].WriteOnly)
	require.Equal(t, "number", pet.Properties["weight"].Type)
	require.Empty(t, pet.Properties["weight"].Format)
	require.Equal(t, "integer", pet.Properties["stock"].Type)
	require.Empty(t, pet.Properties["stock"].Format)
}

func TestRenderRejectsUnrepresentableModels(t *testing.T) {
	schemaRef := func(name string) *Schema {
		return &Schema{Ref: "#/components/schemas/" + name}
	}
	base := func() *Document {
		return &Document{
			OpenAPIVersion: "3.1.1",
			Title:          "Strict",
			APIVersion:     "1.0.0",
			Tags:           []string{"strict"},
			Components: Components{Schemas: []NamedSchema{
				{Name: "Result", GoName: "Result", Schema: &Schema{Type: "string"}},
			}},
			Operations: []Operation{{
				Method: "GET", Path: "/strict", OperationID: "strict.get", GoName: "StrictGet",
				Tags: []string{"strict"}, Responses: []StatusResponse{{
					Status: "200", Response: Response{Schema: schemaRef("Result"), ContentType: "application/json"},
				}},
			}},
		}
	}

	tests := []struct {
		name string
		edit func(*Document)
		want string
	}{
		{name: "invalid package", edit: func(*Document) {}, want: `package name "bad-name" is not a Go identifier`},
		{name: "OpenAPI 2.0 target", edit: func(d *Document) { d.OpenAPIVersion = "2.0" }, want: "only OpenAPI 3.0, 3.1, and 3.2 documents are renderable"},
		{name: "no success response", edit: func(d *Document) { d.Operations[0].Responses[0].Status = "404" }, want: "must define exactly one 2xx response or, when absent, exactly one 3xx response"},
		{name: "multiple success responses", edit: func(d *Document) {
			d.Operations[0].Responses = append(d.Operations[0].Responses, StatusResponse{Status: "201"})
		}, want: "must define exactly one 2xx response or, when absent, exactly one 3xx response"},
		{name: "unresolved schema reference", edit: func(d *Document) {
			d.Operations[0].Responses[0].Response.Schema = schemaRef("Missing")
		}, want: `schema reference "#/components/schemas/Missing" does not resolve`},
		{name: "wrong reference kind", edit: func(d *Document) {
			d.Operations[0].RequestBody = &RequestBody{Ref: "#/components/requestBodies/Body"}
		}, want: "request body references are not renderable"},
		{name: "mixed object and map", edit: func(d *Document) {
			d.Components.Schemas[0].Schema = &Schema{
				Type: "object", Properties: []NamedProperty{{Name: "id", Schema: &Schema{Type: "string"}}},
				AdditionalProperties: &AdditionalProperties{Schema: &Schema{Type: "string"}},
			}
		}, want: "object properties with schema-valued additionalProperties are not renderable"},
		{name: "mixed object and free-form map", edit: func(d *Document) {
			allowed := true
			d.Components.Schemas[0].Schema = &Schema{
				Type: "object", Properties: []NamedProperty{{Name: "id", Schema: &Schema{Type: "string"}}},
				AdditionalProperties: &AdditionalProperties{Allowed: &allowed},
			}
		}, want: "object members with additionalProperties true are not renderable"},
		{name: "unsupported boolean format", edit: func(d *Document) {
			d.Components.Schemas[0].Schema = &Schema{Type: "boolean", Format: "custom"}
		}, want: `boolean format "custom" is not renderable`},
		{name: "response reference", edit: func(d *Document) {
			d.Operations[0].Responses[0].Response = Response{Ref: "#/components/responses/OK"}
		}, want: "response references are not renderable"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := base()
			test.edit(document)
			packageName := "design"
			if test.name == "invalid package" {
				packageName = "bad-name"
			}
			_, err := Render(document, Options{PackageName: packageName})
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestRenderEmitsDefaultAndDeprecatedMetadata(t *testing.T) {
	document := &Document{
		OpenAPIVersion: "3.1.1",
		Title:          "Metadata",
		APIVersion:     "1.0.0",
		Components: Components{Schemas: []NamedSchema{
			{Name: "Widget", GoName: "Widget", Schema: &Schema{
				Type: "object",
				Properties: []NamedProperty{
					{Name: "mouthX", Schema: &Schema{Type: "number", Format: "float", Deprecated: true}},
					{Name: "stable", Schema: &Schema{Type: "string", ReadOnly: true}},
					{Name: "input", Schema: &Schema{Type: "string", WriteOnly: true}},
					{Name: "apiVersion", Schema: &Schema{Type: "string", Default: &SchemaDefault{Value: "1.0"}}},
				},
			}},
		}},
		Operations: []Operation{{
			Method: "GET", Path: "/widgets", OperationID: "widgets.get", GoName: "WidgetsGet",
			Responses: []StatusResponse{{
				Status: "200", Response: Response{
					Schema:      &Schema{Ref: "#/components/schemas/Widget"},
					ContentType: "application/json",
				},
			}},
		}},
	}

	source, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	rendered := string(source)
	require.Contains(t, rendered, `Meta("openapi:deprecated", "true")`)
	require.Contains(t, rendered, `Meta("openapi:readOnly", "true")`)
	require.Contains(t, rendered, `Meta("openapi:writeOnly", "true")`)
	require.Contains(t, rendered, `Default("1.0")`)
}

func TestRenderFallsBackForUnrecognizedAndAbsentFormats(t *testing.T) {
	schemaRef := func(name string) *Schema {
		return &Schema{Ref: "#/components/schemas/" + name}
	}
	document := &Document{
		OpenAPIVersion: "3.1.1",
		Title:          "Formats",
		APIVersion:     "1.0.0",
		Components: Components{Schemas: []NamedSchema{
			{Name: "Record", GoName: "Record", Schema: &Schema{
				Type: "object",
				Properties: []NamedProperty{
					{Name: "birthDate", Schema: &Schema{Type: "string", Format: "dd/mm/yyyy"}},
					{Name: "count", Schema: &Schema{Type: "integer"}},
					{Name: "weight", Schema: &Schema{Type: "integer", Format: "int16"}},
					{Name: "result", Schema: &Schema{Type: "number", Format: ""}},
				},
			}},
		}},
		Operations: []Operation{{
			Method: "GET", Path: "/records", OperationID: "records.get", GoName: "RecordsGet",
			Responses: []StatusResponse{{
				Status: "200", Response: Response{Schema: schemaRef("Record"), ContentType: "application/json"},
			}},
		}},
	}

	source, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	rendered := string(source)
	require.Contains(t, rendered, `Attribute("birthDate", String)`)
	require.NotContains(t, rendered, "FormatDate")
	require.Contains(t, rendered, "Attribute(\"count\", Int, func() {\n\t\tMeta(\"openapi:format\", \"\")")
	require.Contains(t, rendered, `Attribute("weight", Int)`)
	require.Contains(t, rendered, "Attribute(\"result\", Float64, func() {\n\t\tMeta(\"openapi:format\", \"\")")
}

func TestRenderDropsUnrepresentableParameterAndHeaderDeprecation(t *testing.T) {
	document := &Document{
		OpenAPIVersion: "3.1.1",
		Title:          "Deprecated",
		APIVersion:     "1.0.0",
		Operations: []Operation{{
			Method: "GET", Path: "/legacy", OperationID: "legacy.get", GoName: "LegacyGet",
			Parameters: []Parameter{{Name: "q", In: "query", Deprecated: true, Schema: &Schema{Type: "string"}}},
			Responses: []StatusResponse{{
				Status: "200",
				Response: Response{
					ContentType: "application/json",
					Schema:      &Schema{Type: "string"},
					Headers: []NamedHeader{{
						Name:   "X-Legacy",
						Header: Header{Deprecated: true, Schema: &Schema{Type: "string"}},
					}},
				},
			}},
		}},
	}

	source, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.NotContains(t, string(source), "openapi:deprecated")
}

func TestRenderDoesNotMutateDocument(t *testing.T) {
	document, diagnostics, err := Analyze(readFixture(t, "supported.yaml"))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	before, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	after, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.True(t, bytes.Equal(before, after))
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func operationFromImportedSpec(t *testing.T, contract map[string]any, path, method string) map[string]any {
	t.Helper()
	paths, ok := contract["paths"].(map[string]any)
	require.True(t, ok)
	pathItem, ok := paths[path].(map[string]any)
	require.True(t, ok)
	operation, ok := pathItem[method].(map[string]any)
	require.True(t, ok)
	return operation
}
