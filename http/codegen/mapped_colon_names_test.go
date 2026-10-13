package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestMappedColonNamesGenerated preserves complete wire names through codecs and schemas.
func TestMappedColonNamesGenerated(t *testing.T) {
	for _, tc := range []struct {
		name    string
		design  func()
		module  string
		harness string
	}{
		{
			name: "query", design: testdata.MappedColonParamsDSL,
			module:  "example.com/httpmappedparams",
			harness: strings.NewReplacer("?query=x&f=true", "?query%3Afirst=x&query%3Asecond=true").Replace(mappedParamsHarness),
		},
		{
			name: "body", design: testdata.MappedColonBodyDSL,
			module:  "example.com/httpmappedbody",
			harness: strings.NewReplacer(`"n":`, `"account:name":`, `"ag":`, `"account:age":`, `"x":`, `"pair:first":`, `"y":`, `"pair:second":`).Replace(mappedExplicitBodyHarness),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.design)
			doc := parseOpenAPIV3Document(t, renderOpenAPIJSON(t, OpenAPIFiles, root))
			if tc.name == "query" {
				params := doc.Paths.PathItems.GetOrZero("/items/{k}/{id}").Get.Parameters
				names := make(map[string]string)
				for _, parameter := range params {
					names[parameter.Name] = parameter.In
				}
				require.Equal(t, map[string]string{"k": "path", "id": "path", "query:first": "query", "query:second": "query"}, names)
			} else {
				operation := doc.Paths.PathItems.GetOrZero("/accounts/{id}").Post
				request := operation.RequestBody.Content.GetOrZero("application/json").Schema.Schema()
				require.Equal(t, []string{"account:name"}, request.Required)
				require.NotNil(t, request.Properties.GetOrZero("account:name"))
				require.NotNil(t, request.Properties.GetOrZero("account:age"))
				responseRef := operation.Responses.Codes.GetOrZero("200").Content.GetOrZero("application/json").Schema
				require.Equal(t, "#/components/schemas/MappedbodycreateResponseBody", responseRef.GetReference())
				response := doc.Components.Schemas.GetOrZero("MappedbodycreateResponseBody").Schema()
				require.Equal(t, []string{"account:name"}, response.Required)
				require.NotNil(t, response.Properties.GetOrZero("account:name"))
				require.NotNil(t, response.Properties.GetOrZero("account:age"))
			}
			dir := t.TempDir()
			renderHTTPModule(t, dir, tc.module, root)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "mapped_test.go"), []byte(tc.harness), 0o600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "vet", "./...")
			runGoCommand(t, dir, "test", "-race", "-count=1", ".")
		})
	}
}
