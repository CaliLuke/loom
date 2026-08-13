package generator

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/eval"
	"github.com/stretchr/testify/require"
)

func TestGenerateDebugDiagnostics(t *testing.T) {
	t.Cleanup(func() { generatorLoader = generators })

	generatorLoader = func(cmd string) ([]genfunc, error) {
		return []genfunc{
			func(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
				f := &codegen.File{Path: filepath.Join(codegen.Gendir, "types", "diagnostics.go")}
				f.Sections = []codegen.Section{
					codegen.Header("Types", "types", nil),
					codegen.NewRawSection("type-def", "type Diagnostics struct{}\n"),
				}
				return []*codegen.File{f}, nil
			},
		}, nil
	}

	dir := t.TempDir()
	stderr, err := captureGeneratorStderr(t, func() error {
		outputs, genErr := Generate(dir, "gen", true)
		require.NoError(t, genErr)
		require.NotEmpty(t, outputs)
		return nil
	})

	require.NoError(t, err)
	for _, want := range []string{
		"[loom-debug]",
		"stage=load-roots",
		"stage=compute-gen-package",
		"stage=load-generators",
		"stage=prepare-plugins",
		"stage=generate-initial-files",
		"stage=post-generation-plugins",
		"stage=merge-files",
		"stage=write-files",
		"stage=compute-outputs",
		"stage=total",
	} {
		require.Contains(t, stderr, want)
	}
	require.NotContains(t, stderr, "stage=write-file ")
}

func TestGenerateEmitsDeduplicatedWarningsWithoutDebug(t *testing.T) {
	t.Cleanup(func() { generatorLoader = generators })

	generatorLoader = func(cmd string) ([]genfunc, error) {
		return []genfunc{
			func(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
				warnings := []string{
					`OpenAPI 3.1 omits unsupported method QUERY from path "/mixed"`,
					`OpenAPI 3.1 omits unsupported method CONNECT from path "/tunnel" and removes the path because no compatible operations remain`,
				}
				return []*codegen.File{
					{
						Path:     filepath.Join(codegen.Gendir, "openapi.json"),
						Sections: []codegen.Section{codegen.NewRawSection("openapi", "{}")},
						Warnings: warnings,
					},
					{
						Path:     filepath.Join(codegen.Gendir, "openapi.yaml"),
						Sections: []codegen.Section{codegen.NewRawSection("openapi", "openapi: 3.1.0")},
						Warnings: warnings,
					},
				}, nil
			},
		}, nil
	}

	stderr, err := captureGeneratorStderr(t, func() error {
		_, genErr := Generate(t.TempDir(), "gen", false)
		return genErr
	})

	require.NoError(t, err)
	require.Equal(t, "[loom-warning] "+
		`OpenAPI 3.1 omits unsupported method CONNECT from path "/tunnel" and removes the path because no compatible operations remain`+"\n"+
		"[loom-warning] "+
		`OpenAPI 3.1 omits unsupported method QUERY from path "/mixed"`+"\n", stderr)
}

func TestGenerateWrapsWriteFailuresWithStageAndPath(t *testing.T) {
	t.Cleanup(func() { generatorLoader = generators })

	generatorLoader = func(cmd string) ([]genfunc, error) {
		return []genfunc{
			func(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
				f := &codegen.File{Path: filepath.Join(codegen.Gendir, "types", "broken.go")}
				f.Sections = []codegen.Section{
					codegen.Header("Types", "types", nil),
					codegen.NewRawSection("type-def", "type Broken struct{}\n"),
				}
				f.FinalizeFunc = func(path string) error {
					return errors.New("finalize failed")
				}
				return []*codegen.File{f}, nil
			},
		}, nil
	}

	dir := t.TempDir()
	_, err := Generate(dir, "gen", true)
	require.Error(t, err)
	require.ErrorContains(t, err, "stage write-files")
	require.ErrorContains(t, err, filepath.Join(codegen.Gendir, "types", "broken.go"))
	require.ErrorContains(t, err, "finalize failed")
}

func captureGeneratorStderr(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	origStderr := os.Stderr
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = writer
	defer func() {
		os.Stderr = origStderr
	}()

	runErr := fn()

	require.NoError(t, writer.Close())

	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())

	return buf.String(), runErr
}
