package loomsource

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	t.Run("implicit local mode uses repository root", func(t *testing.T) {
		repoRoot := newGitRepository(t)
		require.NoError(t, SetMode(repoRoot, ModeLocal, ""))

		path, err := Resolve(repoRoot, filepath.Join(t.TempDir(), "checkout"))

		require.NoError(t, err)
		require.Equal(t, repoRoot, path)
	})

	t.Run("configured local mode uses its explicit checkout", func(t *testing.T) {
		repoRoot := newGitRepository(t)
		localRoot := newGitRepository(t)
		require.NoError(t, SetMode(repoRoot, ModeLocal, localRoot))

		path, err := Resolve(repoRoot, filepath.Join(t.TempDir(), "checkout"))

		require.NoError(t, err)
		require.Equal(t, localRoot, path)
	})

	t.Run("LOOM_DIR overrides remote mode for one run", func(t *testing.T) {
		repoRoot := newGitRepository(t)
		localRoot := newGitRepository(t)
		require.NoError(t, SetMode(repoRoot, ModeRemote, ""))
		t.Setenv("LOOM_DIR", localRoot)

		path, err := Resolve(repoRoot, filepath.Join(t.TempDir(), "checkout"))

		require.NoError(t, err)
		require.Equal(t, localRoot, path)
	})

	t.Run("invalid LOOM_DIR fails explicitly", func(t *testing.T) {
		repoRoot := newGitRepository(t)
		t.Setenv("LOOM_DIR", filepath.Join(t.TempDir(), "missing"))

		_, err := Resolve(repoRoot, filepath.Join(t.TempDir(), "checkout"))

		require.ErrorContains(t, err, "LOOM_DIR")
	})

	t.Run("remote checkout failure never falls back to working tree", func(t *testing.T) {
		repoRoot := newGitRepository(t)
		require.NoError(t, SetMode(repoRoot, ModeRemote, ""))

		path, err := Resolve(repoRoot, filepath.Join(t.TempDir(), "checkout"))

		require.Empty(t, path)
		require.ErrorContains(t, err, "remote")
	})

	t.Run("remote mode checks out the exact current commit", func(t *testing.T) {
		repoRoot := newGitRepository(t)
		remoteRoot := filepath.Join(t.TempDir(), "origin.git")
		runTestGit(t, "", "init", "--bare", remoteRoot)
		runTestGit(t, repoRoot, "remote", "add", "origin", remoteRoot)
		runTestGit(t, repoRoot, "push", "origin", "HEAD")
		require.NoError(t, SetMode(repoRoot, ModeRemote, ""))
		checkoutDir := filepath.Join(t.TempDir(), "checkout")

		path, err := Resolve(repoRoot, checkoutDir)

		require.NoError(t, err)
		require.Equal(t, checkoutDir, path)
		require.Equal(
			t,
			strings.TrimSpace(runTestGit(t, repoRoot, "rev-parse", "HEAD")),
			strings.TrimSpace(runTestGit(t, checkoutDir, "rev-parse", "HEAD")),
		)
	})
}

func TestSetModeStoresWorktreeLocalConfigurationOutsideTrackedFiles(t *testing.T) {
	repoRoot := newGitRepository(t)
	require.NoError(t, SetMode(repoRoot, ModeLocal, ""))

	path, err := ModePath(repoRoot)
	require.NoError(t, err)
	require.Contains(t, filepath.ToSlash(path), "/.git/")

	status := runTestGit(t, repoRoot, "status", "--porcelain")
	require.Empty(t, strings.TrimSpace(status))
}

func TestModeIsIsolatedPerGitWorktree(t *testing.T) {
	repoRoot := newGitRepository(t)
	worktreeRoot := filepath.Join(t.TempDir(), "linked")
	runTestGit(t, repoRoot, "worktree", "add", "-b", "linked-test", worktreeRoot)
	require.NoError(t, SetMode(repoRoot, ModeLocal, ""))

	mainConfig, err := ReadMode(repoRoot)
	require.NoError(t, err)
	linkedConfig, err := ReadMode(worktreeRoot)
	require.NoError(t, err)

	require.Equal(t, ModeLocal, mainConfig.Mode)
	require.Equal(t, ModeRemote, linkedConfig.Mode)
	mainPath, err := ModePath(repoRoot)
	require.NoError(t, err)
	linkedPath, err := ModePath(worktreeRoot)
	require.NoError(t, err)
	require.NotEqual(t, mainPath, linkedPath)
}

func TestReadModeRejectsMalformedConfiguration(t *testing.T) {
	repoRoot := newGitRepository(t)
	path, err := ModePath(repoRoot)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte("local one two\n"), 0o600))

	_, err = ReadMode(repoRoot)
	require.ErrorContains(t, err, "invalid Loom source mode")
}

func newGitRepository(t *testing.T) string {
	t.Helper()

	repoRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/loom\n"), 0o644))
	runTestGit(t, repoRoot, "init")
	runTestGit(t, repoRoot, "config", "user.email", "loom@example.com")
	runTestGit(t, repoRoot, "config", "user.name", "Loom Test")
	runTestGit(t, repoRoot, "add", "go.mod")
	runTestGit(t, repoRoot, "commit", "-m", "initial")
	return repoRoot
}

func runTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", out)
	return string(out)
}
