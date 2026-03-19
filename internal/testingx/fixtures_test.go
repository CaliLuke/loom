package testingx

import (
	"os"
	"path/filepath"
	"strings"
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

go 1.25.0

require goa.design/goa/v3 v3.0.0

replace goa.design/goa/v3 => ../stale
`
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "go.mod"), []byte(goMod), 0o644))

	repoRoot := filepath.Clean(filepath.Join(workDir, "..", "repo-root"))
	require.NoError(t, PinLocalReplace(workDir, repoRoot))

	updated, err := os.ReadFile(filepath.Join(workDir, "go.mod"))
	require.NoError(t, err)
	require.Contains(t, string(updated), "replace goa.design/goa/v3 => "+repoRoot)
	require.NotContains(t, string(updated), "../stale")
}

func TestRepoRootReturnsRepositoryTopLevel(t *testing.T) {
	t.Parallel()

	root := RepoRoot()
	require.True(t, filepath.IsAbs(root))
	require.True(t, strings.HasSuffix(root, string(filepath.Separator)+"goa-light"))

	info, err := os.Stat(filepath.Join(root, "go.mod"))
	require.NoError(t, err)
	require.False(t, info.IsDir())
}
