package testingx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// RequireTreeMatches asserts that gotDir contains exactly the same regular
// files as wantDir and that each file has identical contents.
func RequireTreeMatches(t testing.TB, wantDir string, gotDir string) {
	t.Helper()

	wantFiles := collectTreeFiles(t, wantDir)
	gotFiles := collectTreeFiles(t, gotDir)
	require.Equal(t, wantFiles, gotFiles)

	for rel := range wantFiles {
		wantContent, err := os.ReadFile(filepath.Join(wantDir, rel))
		require.NoError(t, err)
		gotContent, err := os.ReadFile(filepath.Join(gotDir, rel))
		require.NoError(t, err)
		require.Equal(t, string(wantContent), string(gotContent), "mismatched file %s", rel)
	}
}

func collectTreeFiles(t testing.TB, dir string) map[string]struct{} {
	t.Helper()

	files := map[string]struct{}{}
	require.NoError(t, filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files[rel] = struct{}{}
		return nil
	}))
	return files
}
