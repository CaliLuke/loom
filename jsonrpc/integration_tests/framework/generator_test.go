package framework

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratorRepoRootReplacePrefersLOOMDirEnv(t *testing.T) {
	repoRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/loom\n"), 0o644))
	t.Setenv("LOOM_DIR", repoRoot)

	g := NewGenerator(t.TempDir(), nil)
	path, err := g.repoRootReplace()
	require.NoError(t, err)
	require.Equal(t, repoRoot, path)
}

func TestRenderDesignWritesLoomReplaceDirective(t *testing.T) {
	repoRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/loom\n"), 0o644))
	t.Setenv("LOOM_DIR", repoRoot)

	workDir := t.TempDir()
	g := NewGenerator(workDir, nil)

	require.NoError(t, g.renderDesign(g.buildDesignData()))

	content, err := os.ReadFile(filepath.Join(workDir, "go.mod"))
	require.NoError(t, err)
	require.Contains(t, string(content), "replace github.com/CaliLuke/loom => "+repoRoot)
	require.NotContains(t, string(content), "<no value>")
}
