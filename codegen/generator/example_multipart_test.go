package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestExampleMultipartCompile generates the service, transport, and example
// output for automatic object and custom map multipart methods, then builds
// and vets the module. Only custom codecs belong in example multipart.go, and
// the example CLI must pass the required custom client encoder.
func TestExampleMultipartCompile(t *testing.T) {
	source := loomModuleSource(t)
	codegen.RunDSL(t, multipartExampleDesign)
	dir := t.TempDir()
	goMod := fmt.Sprintf("module example.com/upload\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", source)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))

	_, err := Generate(dir, "gen", "github.com/CaliLuke/loom/codegen/generator", false)
	require.NoError(t, err)
	_, err = Generate(dir, "example", "github.com/CaliLuke/loom/codegen/generator", false)
	require.NoError(t, err)

	multipart, err := os.ReadFile(filepath.Join(dir, "multipart.go"))
	require.NoError(t, err)
	for _, want := range []string{
		"func NotesUploadDecoderFunc(mr *multipart.Reader, p *map[string]string) error",
		"func NotesUploadEncoderFunc(mw *multipart.Writer, p map[string]string) error",
	} {
		require.Contains(t, string(multipart), want)
	}
	require.NotContains(t, string(multipart), "FilesUploadDecoderFunc")
	require.NotContains(t, string(multipart), "FilesUploadEncoderFunc")
	require.NotContains(t, string(multipart), `"example.com/upload/gen/files"`)

	cli, err := os.ReadFile(filepath.Join(dir, "cmd", "upload-cli", "http.go"))
	require.NoError(t, err)
	require.Contains(t, string(cli), "upload.NotesUploadEncoderFunc")
	require.NotContains(t, string(cli), "FilesUploadEncoderFunc")

	_, err = testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err)
	output, err := testingx.RunCmd(dir, "go", "build", "./...")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "vet", "./...")
	require.NoError(t, err, output)
}

// multipartExampleDesign declares a "files" service whose "upload" method
// uses generated multipart codecs, and a "notes" service whose "upload"
// method needs an application-provided multipart decoder.
func multipartExampleDesign() {
	dsl.API("upload", func() {})
	dsl.Service("notes", func() {
		dsl.Method("upload", func() {
			dsl.Payload(dsl.MapOf(dsl.String, dsl.String))
			dsl.Result(dsl.String)
			dsl.HTTP(func() {
				dsl.POST("/notes")
				dsl.MultipartRequest()
			})
		})
	})
	dsl.Service("files", func() {
		dsl.Method("upload", func() {
			dsl.Payload(func() {
				dsl.Attribute("name", dsl.String)
				dsl.Attribute("content", dsl.Bytes)
				dsl.Required("name", "content")
			})
			dsl.Result(dsl.String)
			dsl.HTTP(func() {
				dsl.POST("/upload")
				dsl.MultipartRequest()
			})
		})
	})
}
