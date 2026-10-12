package codegen

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type createDuringRender struct {
	path string
}

func (s createDuringRender) SectionName() string {
	return "racing-creator"
}
func (s createDuringRender) Write(w io.Writer) error {
	if err := os.WriteFile(s.path, []byte("user"), 0o600); err != nil {
		return err
	}
	_, err := io.WriteString(w, "template")
	return err
}

func TestFileSkipExistPreservesRacingCreator(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.txt")
	f := &File{
		Path:      "custom.txt",
		SkipExist: true,
		Sections:  []Section{createDuringRender{path}},
	}
	output, err := f.Render(dir)
	require.NoError(t, err)
	require.Empty(t, output)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "user", string(got))
}

func TestFileSkipExistPreservesDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.txt")
	target := filepath.Join(dir, "missing.txt")
	require.NoError(t, os.Symlink(target, path))
	f := &File{
		Path:      "custom.txt",
		SkipExist: true,
		Sections:  []Section{NewRawSection("content", "template")},
	}
	output, err := f.Render(dir)
	require.NoError(t, err)
	require.Empty(t, output)
	require.NoFileExists(t, target)
	got, err := os.Readlink(path)
	require.NoError(t, err)
	require.Equal(t, target, got)
}

func TestFileSkipExistEntries(t *testing.T) {
	for _, kind := range []string{"absent", "file", "directory", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "custom.txt")
			switch kind {
			case "file":
				require.NoError(t, os.WriteFile(path, []byte("user"), 0o600))
			case "directory":
				require.NoError(t, os.Mkdir(path, 0o750))
			case "symlink":
				require.NoError(t, os.WriteFile(filepath.Join(dir, "target"), []byte("user"), 0o600))
				require.NoError(t, os.Symlink("target", path))
			}
			finalized := false
			f := &File{Path: "custom.txt", SkipExist: true, Sections: []Section{NewRawSection("content", "template")}, FinalizeFunc: func(string) error {
				finalized = true
				return nil
			}}
			got, err := f.Render(dir)
			require.NoError(t, err)
			require.Equal(t, kind == "absent", finalized)
			if kind == "absent" {
				require.Equal(t, path, got)
			} else {
				require.Empty(t, got)
			}
		})
	}
}

func TestFileSkipExistRejectsGenerationTree(t *testing.T) {
	dir := t.TempDir()
	file := &File{
		Path:      "gen/custom.txt",
		SkipExist: true,
		Sections:  []Section{NewRawSection("content", "template")},
	}
	_, err := file.Render(dir)
	require.EqualError(t, err, "create-only artifact gen/custom.txt must be outside gen/")
	require.NoDirExists(t, filepath.Join(dir, "gen"))
}

func TestFileSkipExistAllowsOtherDirectories(t *testing.T) {
	for _, path := range []string{"genesis/custom.txt", "nested/gen/custom.txt"} {
		t.Run(path, func(t *testing.T) {
			dir := t.TempDir()
			file := &File{
				Path:      path,
				SkipExist: true,
				Sections:  []Section{NewRawSection("content", "template")},
			}
			got, err := file.Render(dir)
			require.NoError(t, err)
			require.Equal(t, filepath.Join(dir, path), got)
		})
	}
}

func TestFileSkipExistRejectsLiveAlias(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "gen"), 0o750))
	require.NoError(t, os.Symlink("gen", filepath.Join(dir, "alias")))
	file := &File{
		Path:      "alias/missing/custom.txt",
		SkipExist: true,
		Sections:  []Section{NewRawSection("content", "template")},
	}
	_, err := file.Render(dir)
	require.ErrorContains(t, err, "must be outside gen/")
	require.NoDirExists(t, filepath.Join(dir, "gen", "missing"))
}
