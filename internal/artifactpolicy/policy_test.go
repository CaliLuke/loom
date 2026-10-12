package artifactpolicy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateLiveOwnership(t *testing.T) {
	for _, treeExists := range []bool{false, true} {
		for _, name := range []string{"GEN", "Gen", "gen"} {
			t.Run(name+map[bool]string{true: "-existing", false: "-absent"}[treeExists], func(t *testing.T) {
				dir := t.TempDir()
				if treeExists {
					require.NoError(t, os.Mkdir(filepath.Join(dir, "gen"), 0o750))
				}
				require.ErrorContains(t, Validate(dir, filepath.Join(name, "missing", "custom.txt"), true), "must be outside gen/")
				require.NoError(t, Validate(dir, filepath.Join(name, "custom.txt"), false))
			})
		}
	}
}

func TestValidateSymlinkTargetTraversal(t *testing.T) {
	for _, suffixExists := range []bool{false, true} {
		t.Run(map[bool]string{true: "existing", false: "missing"}[suffixExists], func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "gen", "deeper"), 0o750))
			if suffixExists {
				require.NoError(t, os.Mkdir(filepath.Join(dir, "gen", "branch"), 0o750))
			}
			require.NoError(t, os.Symlink("gen/deeper", filepath.Join(dir, "route")))
			require.NoError(t, os.Symlink("route/../branch", filepath.Join(dir, "alias")))
			require.ErrorContains(t, Validate(dir, "alias/custom.txt", true), "must be outside gen/")
		})
	}
}

func TestValidatePermitsExternalOwnership(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "gen"), 0o750))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "external"), 0o750))
	require.NoError(t, os.Symlink("external", filepath.Join(dir, "alias")))
	require.NoError(t, os.Symlink("gen", filepath.Join(dir, "existing-entry")))
	for _, path := range []string{"genesis/custom.txt", "nested/gen/custom.txt", "alias/missing/custom.txt", "existing-entry"} {
		t.Run(path, func(t *testing.T) {
			require.NoError(t, Validate(dir, path, true))
		})
	}
}

func TestValidateRejectsUnresolvableAncestor(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Symlink("loop", filepath.Join(dir, "loop")))
	require.Error(t, Validate(dir, "loop/custom.txt", true))
	require.NoDirExists(t, filepath.Join(dir, "gen"))
}

func TestValidateUsesFilesystemIdentity(t *testing.T) {
	dir := t.TempDir()
	actual := filepath.Join(dir, "external")
	require.NoError(t, os.Mkdir(actual, 0o750))
	info, err := os.Stat(actual)
	require.NoError(t, err)
	alternate, err := os.Stat(filepath.Join(dir, "EXTERNAL"))
	if os.IsNotExist(err) {
		t.Skip("filesystem distinguishes directory case")
	}
	require.NoError(t, err)
	require.True(t, os.SameFile(info, alternate))
	require.NoError(t, os.Symlink("external", filepath.Join(dir, "gen")))
	require.NoError(t, os.Symlink("EXTERNAL", filepath.Join(dir, "alias")))
	require.ErrorContains(t, Validate(dir, "alias/missing/custom.txt", true), "must be outside gen/")
}

func TestValidateReservesAbsentTreeThroughBaseCaseAlias(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "output")
	require.NoError(t, os.Mkdir(base, 0o750))
	alternate := filepath.Join(dir, "OUTPUT")
	if _, err := os.Stat(alternate); os.IsNotExist(err) {
		t.Skip("filesystem distinguishes directory case")
	} else {
		require.NoError(t, err)
	}
	require.NoError(t, os.Symlink(filepath.Join(alternate, "GEN"), filepath.Join(base, "alias")))
	require.ErrorContains(t, Validate(base, "alias/missing/custom.txt", true), "must be outside gen/")
	require.NoDirExists(t, filepath.Join(base, "gen"))
}
