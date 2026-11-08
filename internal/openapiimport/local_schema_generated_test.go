package openapiimport

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestLocalSchemaTargetsGenerated(t *testing.T) {
	cases := []struct{ name, pointer, kind string }{
		{"property", "#/components/schemas/Container/properties/id", "string"},
		{"alternative", "#/components/schemas/Choice/oneOf/0", "object"},
		{"response", "#/paths/~1inline/get/responses/200/content/application~1json/schema", "string"},
		{"recursive", "#/components/schemas/Container/properties/node", "object"},
		{"collision", "#/components/schemas/Holder/properties/foo-bar", "string"},
	}
	entry := string(localSchemaTargetSource("#/components/schemas/Container")) + "    foo_bar: {type: string}\n    Holder: {type: object, properties: {foo-bar: {type: string}}}\n"
	var paths strings.Builder
	for _, tc := range cases {
		fmt.Fprintf(&paths, "  /%s:\n    get:\n      operationId: %s\n      responses:\n        '200': {description: OK, content: {application/json: {schema: {$ref: '%s'}}}}\n", tc.name, tc.name, tc.pointer)
	}
	entry = strings.Replace(entry, "paths:\n", "paths:\n"+paths.String(), 1)
	document, diagnostics, err := Analyze([]byte(entry))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	for _, schema := range document.Components.Schemas {
		if schema.Name == "foo_bar" {
			require.Equal(t, "FooBar", schema.GoName)
		}
		if schema.Name == "foo-bar" {
			require.Equal(t, "FooBar2", schema.GoName)
		}
	}
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	moduleDir := requireRenderedDesignGenerates(t, rendered)
	spec, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	parsed, err := libopenapi.NewDocument(spec)
	require.NoError(t, err)
	model, err := parsed.BuildV3Model()
	require.NoError(t, err)
	for _, tc := range cases {
		proxy := model.Model.Paths.PathItems.GetOrZero("/" + tc.name).Get.Responses.Codes.GetOrZero("200").Content.GetOrZero("application/json").Schema
		schema := proxy.Schema()
		require.Equal(t, []string{tc.kind}, schema.Type)
		if tc.name == "alternative" {
			require.Equal(t, []string{"name"}, schema.Required)
			require.Equal(t, []string{"string"}, schema.Properties.GetOrZero("name").Schema().Type)
		}
		if tc.name == "recursive" {
			require.NotEmpty(t, proxy.GetReference())
			require.Equal(t, proxy.GetReference(), schema.Properties.GetOrZero("next").GetReference())
		}
	}
}
