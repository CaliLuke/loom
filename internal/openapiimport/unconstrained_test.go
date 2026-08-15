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
    get:
      operationId: getContainer
      responses:
        "200":
          description: container
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Container'}
  /echo:
    post:
      operationId: echo
      requestBody:
        content:
          application/json:
            schema: {}
      responses:
        "200":
          description: echoed
          content:
            application/json:
              schema: {}
components:
  schemas:
    Anything: {}
    Container:
      type: object
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

	require.Len(t, document.Operations, 2)
	echo := requireNormalizedOperation(t, document, "Echo")
	require.NotNil(t, echo.RequestBody)
	require.True(t, echo.RequestBody.Schema.Unconstrained)
	require.True(t, echo.Responses[0].Response.Schema.Unconstrained)
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
	require.Contains(t, design, `Attribute("value", Any)`)
	require.Contains(t, design, `Attribute("values", ArrayOf(Any))`)
	require.Contains(t, design, `Attribute("body", Any)`)
	require.Contains(t, design, `Result(Any)`)

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

func (echoService) GetContainer(context.Context) (*echo.Container, error) {
	return &echo.Container{}, nil
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
`
