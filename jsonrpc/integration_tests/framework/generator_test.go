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
		require.NoError(t, os.WriteFile(path, []byte("local /tmp/goa-light\n"), 0o644))
		require.Equal(t, "/tmp/goa-light", localGoaSourceFromModeFile(path))
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
	t.Setenv("GOA_REPO", "/tmp/goa-light-override")

	g := NewGenerator(t.TempDir(), nil)
	require.Equal(t, "/tmp/goa-light-override", g.repoRootReplace())
}
