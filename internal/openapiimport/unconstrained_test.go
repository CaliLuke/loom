package openapiimport

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

const unconstrainedSchemaSource = `openapi: 3.1.1
info: {title: Echo, version: "1"}
paths:
  /container:
    post:
      operationId: echoContainer
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Container'}
      responses:
        "200":
          description: container
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Container'}
  /direct:
    post:
      operationId: direct
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Anything'}
      responses:
        "200":
          description: direct
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Anything'}
  /echo:
    post:
      operationId: echo
      requestBody:
        required: true
        content:
          application/json:
            schema: {}
      responses:
        "200":
          description: echoed
          content:
            application/json:
              schema: {}
  /failure:
    get:
      operationId: failure
      responses:
        "200":
          description: success
          content:
            application/json:
              schema: {type: string}
        "400":
          description: arbitrary failure
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Anything'}
        "422":
          description: container failure
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Container'}
  /nil:
    get:
      operationId: nilResult
      responses:
        "200":
          description: explicit null
          content:
            application/json:
              schema: {}
components:
  schemas:
    Anything: {}
    Container:
      type: object
      required: [value]
      properties:
        component: {$ref: '#/components/schemas/Anything'}
        value: {}
        values:
          type: array
          items: {}
`

func TestAnalyzePreservesUnconstrainedSchemas(t *testing.T) {
	document, diagnostics, err := Analyze([]byte(unconstrainedSchemaSource))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Len(t, document.Components.Schemas, 2)
	require.Equal(t, "Anything", document.Components.Schemas[0].Name)
	require.True(t, document.Components.Schemas[0].Schema.Unconstrained)

	container := document.Components.Schemas[1].Schema
	require.Equal(t, "Container", document.Components.Schemas[1].Name)
	require.Len(t, container.Properties, 3)
	require.True(t, container.Properties[1].Schema.Unconstrained)
	require.Equal(t, "array", container.Properties[2].Schema.Type)
	require.NotNil(t, container.Properties[2].Schema.Items)
	require.True(t, container.Properties[2].Schema.Items.Unconstrained)

	require.Len(t, document.Operations, 5)
	echo := requireNormalizedOperation(t, document, "Echo")
	require.NotNil(t, echo.RequestBody)
	require.True(t, echo.RequestBody.Schema.Unconstrained)
	require.True(t, echo.Responses[0].Response.Schema.Unconstrained)
}

func TestRenderPreservesCollidingCanonicalComponentNames(t *testing.T) {
	document, diagnostics, err := Analyze([]byte(`openapi: 3.1.1
info: {title: Collisions, version: "1"}
paths:
  /dash:
    get:
      operationId: dash
      responses:
        "200":
          description: dash
          content:
            application/json:
              schema: {$ref: '#/components/schemas/foo-bar'}
  /underscore:
    get:
      operationId: underscore
      responses:
        "200":
          description: underscore
          content:
            application/json:
              schema: {$ref: '#/components/schemas/foo_bar'}
components:
  schemas:
    foo-bar: {type: string}
    foo_bar: {type: integer}
`))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)

	moduleDir := requireRenderedDesignGenerates(t, rendered)
	generated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	var contract map[string]any
	require.NoError(t, json.Unmarshal(generated, &contract))
	components := requireUnconstrainedMap(t, contract["components"], "components")
	schemas := requireUnconstrainedMap(t, components["schemas"], "component schemas")
	require.Contains(t, schemas, "foo-bar")
	require.Contains(t, schemas, "foo_bar")
}

func TestAnalyzeRejectsConstrainedSchemaWithoutType(t *testing.T) {
	_, diagnostics, err := Analyze([]byte(`openapi: 3.1.1
info: {title: Constrained, version: "1"}
paths: {}
components:
  schemas:
    Positive:
      minimum: 1
`))
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "schema")
}

func TestRenderRoundTripsAndRunsUnconstrainedSchemas(t *testing.T) {
	document, diagnostics, err := Analyze([]byte(unconstrainedSchemaSource))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	design := string(rendered)
	require.Contains(t, design, `var ImportedAnything = Type("Anything", Any, func() {`)
	require.Contains(t, design, `Attribute("value", Any, func() {`)
	require.Contains(t, design, `Meta("struct:field:type", "loom.Nullable[any]", "github.com/CaliLuke/loom/pkg", "loom")`)
	require.Contains(t, design, `Attribute("values", ArrayOf(Any))`)
	require.Contains(t, design, `Attribute("body", Any, func() {`)
	require.Contains(t, design, `Result(Any)`)
	require.Contains(t, design, `Error("Status400", func() {`)

	moduleDir := requireRenderedDesignGenerates(t, rendered)
	generated, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	parsed, err := libopenapi.NewDocument(generated)
	require.NoError(t, err)
	_, err = parsed.BuildV3Model()
	require.NoError(t, err)

	var contract map[string]any
	require.NoError(t, json.Unmarshal(generated, &contract))
	components := requireUnconstrainedMap(t, contract["components"], "components")
	schemas := requireUnconstrainedMap(t, components["schemas"], "component schemas")
	require.Empty(t, requireUnconstrainedMap(t, schemas["Anything"], "Anything schema"))
	container := requireUnconstrainedMap(t, schemas["Container"], "Container schema")
	properties := requireUnconstrainedMap(t, container["properties"], "Container properties")
	require.Empty(t, requireUnconstrainedMap(t, properties["value"], "value schema"))
	values := requireUnconstrainedMap(t, properties["values"], "values schema")
	require.Empty(t, requireUnconstrainedMap(t, values["items"], "values item schema"))

	operation := operationFromImportedSpec(t, contract, "/echo", "post")
	requestBody := requireUnconstrainedMap(t, operation["requestBody"], "request body")
	requestContent := requireUnconstrainedMap(t, requestBody["content"], "request content")
	requestJSON := requireUnconstrainedMap(t, requestContent["application/json"], "request JSON media")
	require.Empty(t, requireUnconstrainedMap(t, requestJSON["schema"], "request schema"))
	responses := requireUnconstrainedMap(t, operation["responses"], "responses")
	success := requireUnconstrainedMap(t, responses["200"], "success response")
	responseContent := requireUnconstrainedMap(t, success["content"], "response content")
	responseJSON := requireUnconstrainedMap(t, responseContent["application/json"], "response JSON media")
	require.Empty(t, requireUnconstrainedMap(t, responseJSON["schema"], "response schema"))

	direct := operationFromImportedSpec(t, contract, "/direct", "post")
	directRequest := requireUnconstrainedMap(t, direct["requestBody"], "direct request body")
	directRequestContent := requireUnconstrainedMap(t, directRequest["content"], "direct request content")
	directRequestJSON := requireUnconstrainedMap(t, directRequestContent["application/json"], "direct request JSON media")
	require.Equal(t, "#/components/schemas/Anything", requireUnconstrainedMap(t, directRequestJSON["schema"], "direct request schema")["$ref"])
	directResponses := requireUnconstrainedMap(t, direct["responses"], "direct responses")
	directSuccess := requireUnconstrainedMap(t, directResponses["200"], "direct success response")
	directResponseContent := requireUnconstrainedMap(t, directSuccess["content"], "direct response content")
	directResponseJSON := requireUnconstrainedMap(t, directResponseContent["application/json"], "direct response JSON media")
	require.Equal(t, "#/components/schemas/Anything", requireUnconstrainedMap(t, directResponseJSON["schema"], "direct response schema")["$ref"])

	failure := operationFromImportedSpec(t, contract, "/failure", "get")
	failureResponses := requireUnconstrainedMap(t, failure["responses"], "failure responses")
	badRequest := requireUnconstrainedMap(t, failureResponses["400"], "bad request response")
	badRequestContent := requireUnconstrainedMap(t, badRequest["content"], "bad request content")
	badRequestJSON := requireUnconstrainedMap(t, badRequestContent["application/json"], "bad request JSON media")
	require.Equal(t, "#/components/schemas/Anything", requireUnconstrainedMap(t, badRequestJSON["schema"], "bad request schema")["$ref"])
	unprocessable := requireUnconstrainedMap(t, failureResponses["422"], "unprocessable response")
	unprocessableContent := requireUnconstrainedMap(t, unprocessable["content"], "unprocessable content")
	unprocessableJSON := requireUnconstrainedMap(t, unprocessableContent["application/json"], "unprocessable JSON media")
	require.Equal(t, "#/components/schemas/Container", requireUnconstrainedMap(t, unprocessableJSON["schema"], "unprocessable schema")["$ref"])

	runUnconstrainedRuntimeTests(t, moduleDir)
}

func requireUnconstrainedMap(t *testing.T, value any, name string) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	require.True(t, ok, "%s is %T", name, value)
	return result
}

func requireNormalizedOperation(t *testing.T, document *Document, goName string) *Operation {
	t.Helper()
	for index := range document.Operations {
		if document.Operations[index].GoName == goName {
			return &document.Operations[index]
		}
	}
	require.FailNow(t, "normalized operation not found", goName)
	return nil
}

func runUnconstrainedRuntimeTests(t *testing.T, moduleDir string) {
	t.Helper()
	serviceSource, err := os.ReadFile(filepath.Join(moduleDir, "gen", "echo", "service.go"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(
		filepath.Join(moduleDir, "unconstrained_test.go"),
		[]byte(unconstrainedRuntimeTest),
		0o600,
	))
	command := exec.Command("go", "test", "-mod=mod", "./...")
	command.Dir = moduleDir
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s\n%s", output, serviceSource)
}

const unconstrainedRuntimeTest = `package imported_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	echo "example.com/imported/gen/echo"
	echoserver "example.com/imported/gen/http/echo/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type echoService struct{}

func (echoService) Echo(_ context.Context, payload *echo.EchoPayload) (any, error) {
	return payload.Body, nil
}

func (echoService) EchoContainer(_ context.Context, payload *echo.EchoContainerPayload) (*echo.Container, error) {
	return payload.Body, nil
}

func (echoService) Direct(_ context.Context, payload *echo.DirectPayload) (echo.Anything, error) {
	return echo.Anything(payload.Body), nil
}

func (echoService) Failure(context.Context) (string, error) {
	return "ok", nil
}

func (echoService) NilResult(context.Context) (any, error) {
	return nil, nil
}

func TestUnconstrainedJSONValuesRoundTrip(t *testing.T) {
	endpoints := echo.NewEndpoints(echoService{})
	mux := loomhttp.NewMuxer()
	server := echoserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	echoserver.Mount(mux, server)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)

	tests := []struct {
		name string
		json string
	}{
		{name: "string", json: ` + "`\"value\"`" + `},
		{name: "number", json: "42.5"},
		{name: "object", json: ` + "`{\"key\":\"value\"}`" + `},
		{name: "array", json: ` + "`[1,\"two\",false,null]`" + `},
		{name: "boolean", json: "true"},
		{name: "null", json: "null"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, err := http.NewRequestWithContext(
				t.Context(),
				http.MethodPost,
				httpServer.URL+"/echo",
				bytes.NewBufferString(test.json),
			)
			require.NoError(t, err)
			request.Header.Set("Content-Type", "application/json")
			response, err := httpServer.Client().Do(request)
			require.NoError(t, err)
			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			require.Equal(t, http.StatusOK, response.StatusCode, string(body))
			require.JSONEq(t, test.json, string(body))
		})
	}
}

func TestRequiredUnconstrainedPropertyAcceptsNull(t *testing.T) {
	endpoints := echo.NewEndpoints(echoService{})
	mux := loomhttp.NewMuxer()
	server := echoserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	echoserver.Mount(mux, server)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)

	request, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		httpServer.URL+"/container",
		bytes.NewBufferString(` + "`{\"component\":null,\"value\":null,\"values\":[null,{\"nested\":true}]}`" + `),
	)
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	response, err := httpServer.Client().Do(request)
	require.NoError(t, err)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusOK, response.StatusCode, string(body))
	require.JSONEq(t, ` + "`{\"component\":null,\"value\":null,\"values\":[null,{\"nested\":true}]}`" + `, string(body))
}

func TestNamedUnconstrainedComponentAcceptsNull(t *testing.T) {
	endpoints := echo.NewEndpoints(echoService{})
	mux := loomhttp.NewMuxer()
	server := echoserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	echoserver.Mount(mux, server)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)

	request, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		httpServer.URL+"/direct",
		bytes.NewBufferString("null"),
	)
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	response, err := httpServer.Client().Do(request)
	require.NoError(t, err)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusOK, response.StatusCode, string(body))
	require.JSONEq(t, "null", string(body))
}

func TestNilUnconstrainedResultEncodesNull(t *testing.T) {
	endpoints := echo.NewEndpoints(echoService{})
	mux := loomhttp.NewMuxer()
	server := echoserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	echoserver.Mount(mux, server)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, httpServer.URL+"/nil", nil)
	require.NoError(t, err)
	response, err := httpServer.Client().Do(request)
	require.NoError(t, err)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusOK, response.StatusCode, string(body))
	require.JSONEq(t, "null", string(body))
}
`
