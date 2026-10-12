package openapiimport

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/CaliLuke/loom/internal/testprocess"
	"github.com/stretchr/testify/require"
)

func TestUniformJSONReadinessImport(t *testing.T) {
	const scalar = "{type: string}"
	const array = "{type: array, items: {type: string}}"
	const scalarMap = "{type: object, additionalProperties: {type: string}}"
	const nestedMap = "{type: object, additionalProperties: {type: object, additionalProperties: {type: string}}}"
	const arrayMap = "{type: object, additionalProperties: {type: array, items: {type: string}}}"
	for _, test := range []struct {
		name, parameter string
		accepted        bool
	}{
		{"scalar implicit", "schema: " + scalar, true},
		{"reserved scalar permission preserved", "schema: " + scalar + ", allowReserved: true", true},
		{"scalar explicit defaults", "schema: " + scalar + ", style: form, explode: true", true},
		{"array implicit", "schema: " + array, true},
		{"array explicit defaults", "schema: " + array + ", style: form, explode: true", true},
		{"array comma rejected", "schema: " + array + ", style: form, explode: false", false},
		{"map form implicit rejected", "schema: " + scalarMap, false},
		{"map form explicit rejected", "schema: " + scalarMap + ", style: form, explode: true", false},
		{"map brackets rejected", "schema: " + scalarMap + ", style: deepObject, explode: true", false},
		{"nested brackets rejected", "schema: " + nestedMap + ", style: deepObject, explode: true", false},
		{"map JSON", "content: {application/json: {schema: " + scalarMap + "}}", true},
		{"nested map JSON", "content: {application/json: {schema: " + nestedMap + "}}", true},
		{"map arrays JSON", "content: {application/json: {schema: " + arrayMap + "}}", true},
		{"fixed query object not admitted", "content: {application/json: {schema: {type: object, properties: {name: {type: string}}}}}", false},
		{"JSON array not repeated scalar codec", "content: {application/json: {schema: " + array + "}}", false},
		{"unsupported media", "content: {application/xml: {schema: " + scalarMap + "}}", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := []byte(fmt.Sprintf(`openapi: 3.2.0
info: {title: query, version: "1"}
paths:
  /search:
    get:
      operationId: search
      parameters:
        - {name: filter, in: query, %s}
      responses:
        "204": {description: ok}
`, test.parameter))
			_, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			if test.accepted {
				require.Empty(t, diagnostics)
			} else {
				require.NotEmpty(t, diagnostics)
				fatal, _ := diagnostics.Classify(true)
				require.NotEmpty(t, fatal, "wire mismatches cannot be accepted with allow-lossy")
			}
		})
	}
}

func TestUniformJSONReadinessFormImport(t *testing.T) {
	for _, test := range []struct {
		name, encoding string
		accepted       bool
	}{
		{"implicit JSON object", "", true},
		{"explicit JSON object", "\n            encoding:\n              labels: {contentType: application/json}", true},
		{"brackets rejected", "\n            encoding:\n              labels: {style: deepObject, explode: true}", false},
		{"flattened form rejected", "\n            encoding:\n              labels: {style: form, explode: true}", false},
		{"style overrides JSON rejected", "\n            encoding:\n              labels: {contentType: application/json, style: form, explode: true}", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := []byte(`openapi: 3.2.0
info: {title: form, version: "1"}
paths:
  /submit:
    post:
      operationId: submit
      requestBody:
        content:
          application/x-www-form-urlencoded:
            schema:
              type: object
              properties:
                labels: {type: object, additionalProperties: {type: string}}` + test.encoding + `
      responses:
        "204": {description: ok}
`)
			_, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			if test.accepted {
				require.Empty(t, diagnostics)
			} else {
				require.NotEmpty(t, diagnostics)
				fatal, _ := diagnostics.Classify(true)
				require.NotEmpty(t, fatal, "wire mismatches cannot be accepted with allow-lossy")
			}
		})
	}
}

func TestUniformJSONReadinessImportedMapCompiles(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	t.Setenv("LOOM_DIR", repoRoot)
	document, diagnostics, err := Analyze([]byte(`openapi: 3.2.0
info: {title: query, version: "1"}
paths:
  /search:
    get:
      operationId: search
      parameters:
        - name: filter
          in: query
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Filter'}
      responses:
        "204": {description: ok}
components:
  schemas:
    Filter: {type: object, additionalProperties: {type: string}}
`))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	source, err := Render(document, Options{})
	require.NoError(t, err)
	require.Contains(t, string(source), `Param("filter")`)
	dir := requireRenderedDesignGenerates(t, source)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "wire_test.go"), []byte(importedJSONWireHarness), 0o600))
	for _, args := range [][]string{
		{"mod", "edit", "-require=github.com/CaliLuke/loom@v1.10.0-alpha.9.0.20261012215933-22e452f21fb3"},
		{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}, {"test", "./..."},
	} {
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		command := testprocess.CommandContext(ctx, "go", args...)
		command.Dir = dir
		output, err := command.CombinedOutput()
		cancel()
		require.NoError(t, err, "go %v: %s", args, output)
	}
}

const importedJSONWireHarness = `package imported_test

import (
	client "example.com/imported/gen/http/query/client"
	server "example.com/imported/gen/http/query/server"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestImportedQueryJSON(t *testing.T) {
	values := url.Values{"filter": {"{\"key\":\"value\"}"}}
	request := httptest.NewRequest("GET", "/search?"+values.Encode(), nil)
	payload, err := server.DecodeSearchRequest(loomhttp.NewMuxer(), nil)(request)
	require.NoError(t, err)
	require.Equal(t, "value", payload.Filter["key"])
	encoded := httptest.NewRequest("GET", "/search", nil)
	require.NoError(t, client.EncodeSearchRequest(nil)(encoded, payload))
	require.Equal(t, values.Encode(), encoded.URL.RawQuery)
}
`
