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
	t.Cleanup(func() { Generators = generators })

	Generators = func(cmd string) ([]Genfunc, error) {
		return []Genfunc{
			func(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
				f := &codegen.File{Path: filepath.Join(codegen.Gendir, "types", "diagnostics.go")}
				f.SectionTemplates = []*codegen.SectionTemplate{
					codegen.Header("Types", "types", nil),
					{Name: "type-def", Source: "type Diagnostics struct{}\n"},
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

func TestGenerateWrapsWriteFailuresWithStageAndPath(t *testing.T) {
	t.Cleanup(func() { Generators = generators })

	Generators = func(cmd string) ([]Genfunc, error) {
		return []Genfunc{
			func(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
				f := &codegen.File{Path: filepath.Join(codegen.Gendir, "types", "broken.go")}
				f.SectionTemplates = []*codegen.SectionTemplate{
					codegen.Header("Types", "types", nil),
					{Name: "type-def", Source: "type Broken struct{}\n"},
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
