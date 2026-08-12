package testingx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCopyTreeCopiesRegularFiles(t *testing.T) {
	t.Parallel()

	srcDir := filepath.Join(t.TempDir(), "src")
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "root.txt"), []byte("root"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "nested", "child.txt"), []byte("child"), 0o600))

	dstDir := filepath.Join(t.TempDir(), "dst")
	require.NoError(t, CopyTree(srcDir, dstDir))

	root, err := os.ReadFile(filepath.Join(dstDir, "root.txt"))
	require.NoError(t, err)
	require.Equal(t, "root", string(root))

	child, err := os.ReadFile(filepath.Join(dstDir, "nested", "child.txt"))
	require.NoError(t, err)
	require.Equal(t, "child", string(child))
}

func TestPinLocalReplaceUpdatesGoMod(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	goMod := `module example.com/fixture

go 1.27rc2

require github.com/CaliLuke/loom v1.0.0

replace github.com/CaliLuke/loom => ../stale
`
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "go.mod"), []byte(goMod), 0o644))

	repoRoot := filepath.Clean(filepath.Join(workDir, "..", "repo-root"))
	require.NoError(t, PinLocalReplace(workDir, repoRoot))

	updated, err := os.ReadFile(filepath.Join(workDir, "go.mod"))
	require.NoError(t, err)
	require.Contains(t, string(updated), "replace github.com/CaliLuke/loom => "+repoRoot)
	require.NotContains(t, string(updated), "../stale")
}

func TestRepoRootReturnsRepositoryTopLevel(t *testing.T) {
	t.Parallel()

	root := RepoRoot()
	require.True(t, filepath.IsAbs(root))

	goModPath := filepath.Join(root, "go.mod")
	info, err := os.Stat(goModPath)
	require.NoError(t, err)
	require.False(t, info.IsDir())

	goMod, err := os.ReadFile(goModPath)
	require.NoError(t, err)
	require.Contains(t, string(goMod), "module github.com/CaliLuke/loom")
}
