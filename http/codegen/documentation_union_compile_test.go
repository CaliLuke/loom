package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestGeneratedDocumentationUnionsCompile(t *testing.T) {
	root := RunHTTPDSL(t, testdata.RawRequestBodyOpenAPIDSL)
	dir := t.TempDir()
	if output := os.Getenv("LOOM_DOCUMENTATION_OUTPUT"); output != "" {
		dir = output
	}
	renderHTTPModule(t, dir, "example.com/documentation", root)
	files, err := OpenAPIFiles(root)
	require.NoError(t, err)
	for _, file := range files {
		_, err := file.Render(dir)
		require.NoError(t, err)
	}
	service, err := os.ReadFile(filepath.Join(dir, "gen", "raw_upload", "service.go"))
	require.NoError(t, err)
	require.NotContains(t, string(service), "type RawUploadResult", "documentation-only response schemas must not generate codecs")
	require.Contains(t, string(service), "type RawUploadChoice", "the separately used typed union still needs a codec")
	if os.Getenv("LOOM_DOCUMENTATION_GENERATE_ONLY") != "" {
		return
	}
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
}
