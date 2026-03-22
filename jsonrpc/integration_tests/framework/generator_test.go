package framework

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalGoaSourceFromModeFile(t *testing.T) {
	t.Run("local mode returns configured path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), goaSourceModeFile)
		require.NoError(t, os.WriteFile(path, []byte("local /tmp/loom\n"), 0o644))
		require.Equal(t, "/tmp/loom", localGoaSourceFromModeFile(path))
	})

	t.Run("remote mode disables local override", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), goaSourceModeFile)
		require.NoError(t, os.WriteFile(path, []byte("remote\n"), 0o644))
		require.Empty(t, localGoaSourceFromModeFile(path))
	})

	t.Run("missing path disables local override", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), goaSourceModeFile)
		require.NoError(t, os.WriteFile(path, []byte("local\n"), 0o644))
		require.Empty(t, localGoaSourceFromModeFile(path))
	})
}

func TestGeneratorRepoRootReplacePrefersGOARepoEnv(t *testing.T) {
	t.Setenv("GOA_REPO", "/tmp/loom-override")

	g := NewGenerator(t.TempDir(), nil)
	require.Equal(t, "/tmp/loom-override", g.repoRootReplace())
}
