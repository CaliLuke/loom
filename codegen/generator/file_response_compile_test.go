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
	tests := []struct {
		name string
		dsl  func()
	}{
		{name: "empty result GET", dsl: fileResponseEmptyResultCompileDSL},
		{name: "inline metadata GET and HEAD", dsl: fileResponseInlineResultCompileDSL},
		{name: "reusable metadata type", dsl: fileResponseUserTypeResultCompileDSL},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testFileResponseGeneratedApplicationCompiles(t, test.dsl)
		})
	}
}

func testFileResponseGeneratedApplicationCompiles(t *testing.T, design func()) {
	t.Helper()
	root := codegen.RunDSL(t, design)
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

func fileResponseEmptyResultCompileDSL() {
	dsl.API("file-response", func() {})
	dsl.Service("files", func() {
		dsl.Method("download", func() {
			dsl.HTTP(func() {
				dsl.GET("/download")
				dsl.FileResponse()
			})
		})
	})
}

func fileResponseInlineResultCompileDSL() {
	dsl.API("file-response", func() {})
	dsl.Service("files", func() {
		dsl.Method("download", func() {
			dsl.Payload(func() {
				dsl.Attribute("id", dsl.String)
				dsl.Required("id")
			})
			dsl.Result(func() {
				dsl.Attribute("etag", dsl.String)
			})
			dsl.HTTP(func() {
				dsl.GET("/download/{id}")
				dsl.HEAD("/download/{id}")
				dsl.FileResponse()
				dsl.Response(func() {
					dsl.ContentType("application/octet-stream")
					dsl.Header("etag:ETag")
				})
			})
		})
	})
}

func fileResponseUserTypeResultCompileDSL() {
	dsl.API("file-response", func() {})
	metadata := dsl.Type("FileMetadata", func() {
		dsl.Attribute("etag", dsl.String)
		dsl.Attribute("disposition", dsl.String)
	})
	dsl.Service("files", func() {
		dsl.Method("download", func() {
			dsl.Result(metadata)
			dsl.HTTP(func() {
				dsl.GET("/download")
				dsl.FileResponse()
				dsl.Response(func() {
					dsl.Header("etag:ETag")
					dsl.Header("disposition:Content-Disposition")
				})
			})
		})
	})
}
