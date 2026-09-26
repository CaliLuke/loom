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
// output for a design with a MultipartRequest method, then builds and vets
// the module. A single example multipart.go must hold the functions of every
// service and reference each payload through the service package import it
// declares, and the example CLI must pass every client multipart encoder.
func TestExampleMultipartCompile(t *testing.T) {
	source := loomModuleSource(t)
	codegen.RunDSL(t, multipartExampleDesign)
	dir := t.TempDir()
	goMod := fmt.Sprintf("module example.com/upload\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", source)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))

	_, err := Generate(dir, "gen", false)
	require.NoError(t, err)
	_, err = Generate(dir, "example", false)
	require.NoError(t, err)

	multipart, err := os.ReadFile(filepath.Join(dir, "multipart.go"))
	require.NoError(t, err)
	for _, want := range []string{
		`files "example.com/upload/gen/files"`,
		"func NotesUploadDecoderFunc(mr *multipart.Reader, p *map[string]string) error",
		"func NotesUploadEncoderFunc(mw *multipart.Writer, p map[string]string) error",
		"func FilesUploadEncoderFunc(mw *multipart.Writer, p *files.UploadPayload) error",
	} {
		require.Contains(t, string(multipart), want)
	}
	require.NotContains(t, string(multipart), "FilesUploadDecoderFunc")

	_, err = testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err)
	output, err := testingx.RunCmd(dir, "go", "build", "./...")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "vet", "./...")
	require.NoError(t, err, output)
}

// multipartExampleDesign declares a "files" service whose "upload" method
// uses a generated multipart decoder, and a "notes" service whose "upload"
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
