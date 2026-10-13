package codegen

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

//go:embed testdata/explicit_empty_body_harness.go.txt
var explicitEmptyBodyHarness string

func TestExplicitEmptyBodyGenerated(t *testing.T) {
	root := RunHTTPDSL(t, testdata.InlineBodySelectionDSL)
	dir := t.TempDir()
	if output := os.Getenv("LOOM_EMPTY_BODY_OUTPUT"); output != "" {
		dir = output
		require.NoError(t, os.MkdirAll(dir, 0o750))
	}
	renderHTTPModule(t, dir, "example.com/emptybody", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "empty_body_test.go"), []byte(explicitEmptyBodyHarness), 0o600))
	if os.Getenv("LOOM_EMPTY_BODY_GENERATE_ONLY") != "" {
		return
	}
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
}

func TestExplicitEmptyBodyOpenAPI(t *testing.T) {
	root := RunHTTPDSL(t, testdata.InlineBodySelectionDSL)
	spec := renderOpenAPIJSON(t, OpenAPIFiles, root)
	if output := os.Getenv("LOOM_EMPTY_BODY_OUTPUT"); output != "" {
		require.NoError(t, os.WriteFile(filepath.Join(output, "openapi.json"), spec, 0o600))
	}
	doc := parseOpenAPIV3Document(t, spec)
	for _, tc := range []struct {
		path   string
		status string
	}{
		{"/empty_selected", "200"}, {"/empty_type", "200"}, {"/empty_error", "400"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			operation := doc.Paths.PathItems.GetOrZero(tc.path).Get
			response := operation.Responses.Codes.GetOrZero(tc.status)
			require.NotNil(t, response)
			require.NotNil(t, response.Content)
			content := response.Content.GetOrZero("application/json")
			require.NotNil(t, content)
			require.Equal(t, "#/components/schemas/EmptyRecord", content.Schema.GetReference())
			schema := doc.Components.Schemas.GetOrZero("EmptyRecord").Schema()
			require.Equal(t, []string{"object"}, schema.Type)
			if schema.Properties != nil {
				require.Zero(t, schema.Properties.Len())
			}
			require.NotNil(t, schema.AdditionalProperties)
			require.False(t, schema.AdditionalProperties.B)
		})
	}
}
