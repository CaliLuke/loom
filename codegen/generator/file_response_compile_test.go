package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestFileResponseGeneratedApplicationCompiles(t *testing.T) {
	root := codegen.RunDSL(t, fileResponseCompileDSL)
	roots := []eval.Root{root}
	genpkg := "example.com/file-response/gen"

	serviceFiles, err := Service(genpkg, roots)
	require.NoError(t, err)
	transportFiles, err := Transport(genpkg, roots)
	require.NoError(t, err)
	exampleFiles, err := Example(genpkg, roots)
	require.NoError(t, err)
	files := mergeFilesByPath(append(append(serviceFiles, transportFiles...), exampleFiles...))

	dir := t.TempDir()
	for _, file := range files {
		_, err := file.Render(dir)
		require.NoError(t, err, file.Path)
	}
	goMod := fmt.Sprintf("module example.com/file-response\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
	_, err = testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err)
	output, err := testingx.RunCmd(dir, "go", "test", "./...")
	require.NoError(t, err, output)
}

func fileResponseCompileDSL() {
	dsl.API("file-response", func() {})
	dsl.Service("files", func() {
		dsl.Method("download", func() {
			dsl.Result(func() {
				dsl.Attribute("etag", dsl.String)
			})
			dsl.HTTP(func() {
				dsl.GET("/download")
				dsl.HEAD("/download")
				dsl.FileResponse()
				dsl.Response(func() {
					dsl.ContentType("application/octet-stream")
					dsl.Header("etag:ETag")
				})
			})
		})
	})
}
