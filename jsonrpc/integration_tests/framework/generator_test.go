package framework

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalLoomSourceFromModeFile(t *testing.T) {
	const modeFileName = ".loom_source_mode"

	t.Run("local mode returns configured path", func(t *testing.T) {
		repoRoot := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/loom\n"), 0o644))
		path := filepath.Join(t.TempDir(), modeFileName)
		require.NoError(t, os.WriteFile(path, []byte("local "+repoRoot+"\n"), 0o644))
		require.Equal(t, repoRoot, localLoomSourceFromModeFile(path, ""))
	})

	t.Run("remote mode disables local override", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), modeFileName)
		require.NoError(t, os.WriteFile(path, []byte("remote\n"), 0o644))
		require.Empty(t, localLoomSourceFromModeFile(path, ""))
	})

	t.Run("implicit local mode uses default source", func(t *testing.T) {
		repoRoot := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/loom\n"), 0o644))
		path := filepath.Join(t.TempDir(), modeFileName)
		require.NoError(t, os.WriteFile(path, []byte("local\n"), 0o644))
		require.Equal(t, repoRoot, localLoomSourceFromModeFile(path, repoRoot))
	})

	t.Run("stale path disables local override", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), modeFileName)
		require.NoError(t, os.WriteFile(path, []byte("local /tmp/does-not-exist\n"), 0o644))
		require.Empty(t, localLoomSourceFromModeFile(path, ""))
	})
}

func TestGeneratorRepoRootReplacePrefersLOOMRepoEnv(t *testing.T) {
	repoRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/loom\n"), 0o644))
	t.Setenv("LOOM_REPO", repoRoot)

	g := NewGenerator(t.TempDir(), nil)
	require.Equal(t, repoRoot, g.repoRootReplace())
}

func TestRenderDesignWritesLoomReplaceDirective(t *testing.T) {
	repoRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/loom\n"), 0o644))
	t.Setenv("LOOM_REPO", repoRoot)

	workDir := t.TempDir()
	g := NewGenerator(workDir, nil)

	require.NoError(t, g.renderDesign(g.buildDesignData()))

	content, err := os.ReadFile(filepath.Join(workDir, "go.mod"))
	require.NoError(t, err)
	require.Contains(t, string(content), "replace github.com/CaliLuke/loom => "+repoRoot)
	require.NotContains(t, string(content), "<no value>")
}
