package openapiimport

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderPreservesUntaggedResponseOneOf(t *testing.T) {
	source := []byte(`openapi: 3.1.0
info: {title: Gap, version: '1'}
paths:
  /items:
    get:
      operationId: getItem
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                oneOf:
                  - type: object
                    additionalProperties: false
                    required: [data]
                    properties:
                      data: {type: string, pattern: '^[a-z]+$'}
                      error: {type: string}
                      note: {type: string}
                  - type: object
                    required: [error]
                    properties:
                      error: {type: string}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Len(t, document.Components.Schemas, 2)
	response := document.Operations[0].Responses[0].Response.Schema
	require.Len(t, response.OneOf, 2)

	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	design := string(rendered)
	require.Contains(t, design, `Result(OneOf(ImportedGetItemResponseData, ImportedGetItemResponseError), func() {`)
	require.Contains(t, design, "Untagged()")
	requireRenderedDesignEvaluates(t, rendered, 1)

	moduleDir := requireRenderedDesignGenerates(t, rendered)
	requireUntaggedUnionRuntime(t, moduleDir)
	contract := readGeneratedOpenAPIContract(t, moduleDir)
	operation := operationFromImportedSpec(t, contract, "/items", "get")
	schema := operation["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	if ref, ok := schema["$ref"].(string); ok {
		schema = referencedSchema(t, contract, ref)
	}
	branches := schema["oneOf"].([]any)
	require.Len(t, branches, 2)
	require.NotContains(t, schema, "discriminator")
	require.Equal(t, []any{"data"}, referencedSchema(t, contract, branches[0].(map[string]any)["$ref"].(string))["required"])
	require.Equal(t, []any{"error"}, referencedSchema(t, contract, branches[1].(map[string]any)["$ref"].(string))["required"])
}

func requireUntaggedUnionRuntime(t *testing.T, moduleDir string) {
	t.Helper()
	serviceDir := filepath.Join(moduleDir, "gen", "gap")
	testSource := []byte(`package gap

import (
	"encoding/json"
	"testing"
)

func TestImportedUntaggedUnionRuntime(t *testing.T) {
	valid := []struct {
		body string
		ok   bool
	}{
		{body: "{\"data\":\"ready\"}", ok: true},
		{body: "{\"error\":\"failed\"}"},
	}
	for _, test := range valid {
		var result GetItemResponseDataOrGetItemResponseError
		if err := json.Unmarshal([]byte(test.body), &result); err != nil {
			t.Errorf("decode %s: %v", test.body, err)
			continue
		}
		if _, ok := result.AsGetItemResponseData(); ok != test.ok {
			t.Errorf("decode %s selected data = %t, want %t", test.body, ok, test.ok)
		}
	}

	for _, body := range []string{
		"{}",
		"{\"data\":\"ready\",\"error\":\"failed\"}",
		"{\"data\":\"123\"}",
		"{\"data\":\"ready\",\"note\":null}",
		"{\"data\":\"ready\",\"extra\":true}",
		"{\"Data\":\"ready\"}",
	} {
		var result GetItemResponseDataOrGetItemResponseError
		if err := json.Unmarshal([]byte(body), &result); err == nil {
			t.Errorf("decode %s unexpectedly succeeded", body)
		}
	}

	encoded, err := json.Marshal(NewGetItemResponseDataOrGetItemResponseErrorGetItemResponseData(
		&GetItemResponseData{Data: "ready"},
	))
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != "{\"data\":\"ready\"}" {
		t.Errorf("encoded %s", encoded)
	}

	var reused GetItemResponseDataOrGetItemResponseError
	if err := json.Unmarshal([]byte("{\"data\":\"ready\"}"), &reused); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte("{\"data\":\"ready\",\"error\":\"failed\"}"), &reused); err == nil {
		t.Fatal("ambiguous decode unexpectedly succeeded")
	}
	if data, ok := reused.AsGetItemResponseData(); !ok || data.Data != "ready" {
		t.Fatalf("failed decode mutated receiver: %#v", reused)
	}
}
`)
	require.NoError(t, os.WriteFile(filepath.Join(serviceDir, "untagged_runtime_test.go"), testSource, 0o600))
	command := exec.Command("go", "test", "-mod=mod", "./gen/gap")
	command.Dir = moduleDir
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}

func TestUntaggedResponseOneOfKeepsUnsupportedShapesRejected(t *testing.T) {
	tests := map[string]string{
		"scalar branch": `oneOf: [{type: string}, {type: integer}]`,
		"single branch": `oneOf: [{type: object, required: [data], properties: {data: {type: string}}}]`,
	}
	for name, schema := range tests {
		t.Run(name, func(t *testing.T) {
			source := []byte("openapi: 3.1.0\ninfo: {title: Gap, version: '1'}\npaths:\n  /items:\n    get:\n      operationId: getItem\n      responses:\n        '200':\n          description: OK\n          content:\n            application/json:\n              schema:\n                " + schema + "\n")
			_, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			requireDiagnosticCode(t, diagnostics, "schema-composition")
		})
	}
}

func TestRenderPreservesReferencedUntaggedRequestOneOf(t *testing.T) {
	source := []byte(`openapi: 3.1.0
info: {title: Commands, version: '1'}
paths:
  /commands:
    post:
      operationId: runCommand
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Command'}
      responses:
        '200':
          description: done
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Command'}
components:
  schemas:
    Start:
      type: object
      required: [start]
      properties: {start: {type: string}}
    Stop:
      type: object
      required: [stop]
      properties: {stop: {type: string}}
    Command:
      oneOf:
        - $ref: '#/components/schemas/Start'
        - $ref: '#/components/schemas/Stop'
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	design := string(rendered)
	require.Contains(t, design, `var ImportedCommand = Type("Command", OneOf(ImportedStart, ImportedStop), func() {`)
	require.Contains(t, design, "Untagged()")
	require.Contains(t, design, `Attribute("body", ImportedCommand)`)
	require.Contains(t, design, `Body("body"`)
	requireRenderedDesignEvaluates(t, rendered, 1)

	moduleDir := requireRenderedDesignGenerates(t, rendered)
	requireNamedUntaggedUnionRuntime(t, moduleDir)
	contract := readGeneratedOpenAPIContract(t, moduleDir)
	command := contract["components"].(map[string]any)["schemas"].(map[string]any)["Command"].(map[string]any)
	require.Len(t, command["oneOf"].([]any), 2)
	require.NotContains(t, command, "discriminator")
	operation := operationFromImportedSpec(t, contract, "/commands", "post")
	requestRef := operation["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)["$ref"]
	responseRef := operation["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)["$ref"]
	require.Equal(t, "#/components/schemas/Command", requestRef)
	require.Equal(t, requestRef, responseRef)
}

func requireNamedUntaggedUnionRuntime(t *testing.T, moduleDir string) {
	t.Helper()
	serviceTest := []byte(`package commands

import (
	"encoding/json"
	"testing"
)

func TestNamedUntaggedUnionRuntime(t *testing.T) {
	var command Command
	if err := json.Unmarshal([]byte("{\"start\":\"now\"}"), &command); err != nil {
		t.Fatal(err)
	}
	if _, ok := command.AsStart(); !ok {
		t.Fatalf("decoded command: %#v", command)
	}
	if err := json.Unmarshal([]byte("{\"Start\":\"later\"}"), &command); err == nil {
		t.Fatal("case-folded JSON member unexpectedly matched")
	}
	if _, ok := command.AsStart(); !ok {
		t.Fatalf("failed decode mutated command: %#v", command)
	}
	var stop Stop
	if err := json.Unmarshal([]byte("{\"stop\":\"now\"}"), &stop); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(NewCommandStop(&stop))
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != "{\"stop\":\"now\"}" {
		t.Fatalf("encoded %s", encoded)
	}
}
`)
	serviceDir := filepath.Join(moduleDir, "gen", "commands")
	require.NoError(t, os.WriteFile(filepath.Join(serviceDir, "named_untagged_runtime_test.go"), serviceTest, 0o600))

	clientTest := []byte(`package client

import (
	"encoding/json"
	"testing"
)

func TestHTTPUntaggedUnionRuntime(t *testing.T) {
	var response RunCommandResponseBody
	if err := json.Unmarshal([]byte("{\"start\":\"now\"}"), &response); err != nil {
		t.Fatal(err)
	}
	if _, ok := response.AsStart(); !ok {
		t.Fatalf("decoded response: %#v", response)
	}
	if err := json.Unmarshal([]byte("{\"Start\":\"later\"}"), &response); err == nil {
		t.Fatal("case-folded JSON member unexpectedly matched")
	}
	if _, ok := response.AsStart(); !ok {
		t.Fatalf("failed decode mutated response: %#v", response)
	}
}
`)
	clientDir := filepath.Join(moduleDir, "gen", "http", "commands", "client")
	require.NoError(t, os.WriteFile(filepath.Join(clientDir, "untagged_runtime_test.go"), clientTest, 0o600))

	command := exec.Command("go", "test", "-mod=mod", "./gen/commands", "./gen/http/commands/client")
	command.Dir = moduleDir
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}

func TestUntaggedOneOfRejectsNestedObjectBranch(t *testing.T) {
	source := []byte(`openapi: 3.1.0
info: {title: Gap, version: '1'}
paths:
  /items:
    get:
      operationId: getItem
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                oneOf:
                  - type: object
                    required: [data]
                    properties:
                      data:
                        type: object
                        properties: {value: {type: string}}
                  - type: object
                    required: [error]
                    properties: {error: {type: string}}
`)

	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "schema-oneof-branch")
}

func TestUntaggedOneOfRejectsStringEncodedParameter(t *testing.T) {
	source := []byte(`openapi: 3.1.0
info: {title: Gap, version: '1'}
paths:
  /items:
    get:
      operationId: getItem
      parameters:
        - name: filter
          in: query
          schema:
            oneOf:
              - type: object
                required: [data]
                properties: {data: {type: string}}
              - type: object
                required: [error]
                properties: {error: {type: string}}
      responses: {'204': {description: done}}
`)

	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "schema-oneof-location")
}

func TestUntaggedOneOfRejectsDeclaredErrorResponse(t *testing.T) {
	source := []byte(`openapi: 3.1.0
info: {title: Gap, version: '1'}
paths:
  /items:
    get:
      operationId: getItem
      responses:
        '204': {description: done}
        '400':
          description: failed
          content:
            application/json:
              schema:
                oneOf:
                  - type: object
                    required: [first]
                    properties: {first: {type: string}}
                  - type: object
                    required: [second]
                    properties: {second: {type: string}}
`)

	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "schema-oneof-location")
}
