package valuecontract

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSourceSnapshotIdentity captures dirty and new files independently of HEAD.
func TestSourceSnapshotIdentity(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "tracked.go"), []byte("package before\n"), 0640))
	paths := []string{"tracked.go"}
	before, err := sourceInventory(root, paths)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "tracked.go"), []byte("package dirty\n"), 0640))
	dirty, err := sourceInventory(root, paths)
	require.NoError(t, err)
	require.NotEqual(t, before, dirty)
	beforeID, err := snapshotIdentity(before)
	require.NoError(t, err)
	dirtyID, err := snapshotIdentity(dirty)
	require.NoError(t, err)
	require.NotEqual(t, beforeID, dirtyID)
	require.NoError(t, os.WriteFile(filepath.Join(root, "new.go"), []byte("package dirty\n"), 0600))
	withNew, err := sourceInventory(root, []string{"tracked.go", "new.go"})
	require.NoError(t, err)
	require.Len(t, withNew, 2)
	destination := filepath.Join(t.TempDir(), "snapshot")
	require.NoError(t, materializeSource(root, destination, withNew))
	require.NoError(t, verifySourceSnapshot(destination, withNew))
	require.NoError(t, os.WriteFile(filepath.Join(root, "tracked.go"), []byte("package later\n"), 0640))
	require.NoError(t, verifySourceSnapshot(destination, withNew), "live edit cannot alter captured replacement source")
	captured, err := os.ReadFile(filepath.Join(destination, "tracked.go"))
	require.NoError(t, err)
	require.Equal(t, "package dirty\n", string(captured))
	info, err := os.Stat(filepath.Join(destination, "tracked.go"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0640), info.Mode().Perm())
	require.NoError(t, os.WriteFile(filepath.Join(destination, "tracked.go"), []byte("package corrupted\n"), 0640))
	require.Error(t, verifySourceSnapshot(destination, withNew))
}

// TestSourceSnapshotRejectsTornCapture checks source drift and escaping links.
func TestSourceSnapshotRejectsTornCapture(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "source.go"), []byte("package first\n"), 0600))
	before, err := sourceInventory(root, []string{"source.go"})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "source.go"), []byte("package second\n"), 0600))
	require.Error(t, materializeSource(root, filepath.Join(t.TempDir(), "snapshot"), before))
	require.NoError(t, os.Symlink("../outside", filepath.Join(root, "escape")))
	_, err = sourceInventory(root, []string{"escape"})
	require.Error(t, err)
	require.NoError(t, os.Symlink("source.go", filepath.Join(root, "inside")))
	safe, err := sourceInventory(root, []string{"source.go", "inside"})
	require.NoError(t, err)
	destination := filepath.Join(t.TempDir(), "snapshot")
	require.NoError(t, materializeSource(root, destination, safe))
	target, err := os.Readlink(filepath.Join(destination, "inside"))
	require.NoError(t, err)
	require.Equal(t, "source.go", target)
}

// TestCommonSpecimenUsesCapturedHelpers verifies package-local imports/resources.
func TestCommonSpecimenUsesCapturedHelpers(t *testing.T) {
	common, revision := t.TempDir(), t.TempDir()
	for _, tc := range []struct{ dir, value string }{{common, "common"}, {revision, "wrong-revision"}} {
		require.NoError(t, os.MkdirAll(filepath.Join(tc.dir, "helper"), 0700))
		require.NoError(t, os.WriteFile(filepath.Join(tc.dir, "go.mod"), []byte("module example.org/specimen\n\ngo 1.27.0\n"), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(tc.dir, "specimen.go"), []byte("package specimen\nimport \"example.org/specimen/helper\"\nfunc Value() string { return helper.Value }\n"), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(tc.dir, "helper", "helper.go"), []byte("package helper\nimport _ \"embed\"\n//go:embed value.txt\nvar Value string\n"), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(tc.dir, "helper", "value.txt"), []byte(tc.value), 0600))
	}
	for _, framework := range []string{common, revision} {
		module := t.TempDir()
		require.NoError(t, copyCommonSpecimen(common, filepath.Join(module, "specimen"), "example.org/specimen", "example.com/probe/specimen"))
		require.NoError(t, os.Remove(filepath.Join(module, "specimen", "go.mod")))
		mod := "module example.com/probe\n\ngo 1.27.0\n\nrequire example.org/specimen v0.0.0\nreplace example.org/specimen => " + framework + "\n"
		require.NoError(t, os.WriteFile(filepath.Join(module, "go.mod"), []byte(mod), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(module, "main.go"), []byte("package main\nimport (\"fmt\";\"example.com/probe/specimen\")\nfunc main(){fmt.Print(specimen.Value())}\n"), 0600))
		got := runPhase(context.Background(), module, []string{"LOOM_DIR=" + framework}, "control", "go", "run", ".")
		require.Zero(t, got.Exit, got.Output)
		require.Equal(t, "common", got.Output)
		copied, err := os.ReadFile(filepath.Join(module, "specimen", "helper", "value.txt"))
		require.NoError(t, err)
		require.Equal(t, "common", string(copied))
	}
}

// TestGitSourceFilesIncludesDirtyAndUntracked exercises the actual Git discovery boundary.
func TestGitSourceFilesIncludesDirtyAndUntracked(t *testing.T) {
	root := t.TempDir()
	initialized := runPhase(t.Context(), root, nil, "control", "git", "init")
	require.Zero(t, initialized.Exit, initialized.Output)
	tracked := filepath.Join(root, "tracked.go")
	require.NoError(t, os.WriteFile(tracked, []byte("package original\n"), 0600))
	staged := runPhase(t.Context(), root, nil, "control", "git", "add", "tracked.go")
	require.Zero(t, staged.Exit, staged.Output)
	before, err := sourceInventory(root, gitSourceFiles(t, root))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tracked, []byte("package dirty\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "new.go"), []byte("package dirty\n"), 0600))
	paths := gitSourceFiles(t, root)
	require.ElementsMatch(t, []string{"tracked.go", "new.go"}, paths)
	after, err := sourceInventory(root, paths)
	require.NoError(t, err)
	beforeID, err := snapshotIdentity(before)
	require.NoError(t, err)
	afterID, err := snapshotIdentity(after)
	require.NoError(t, err)
	require.NotEqual(t, beforeID, afterID)
}

// TestSpecimenRewritePreservesAuthoredStrings separates Go paths from DSL values.
func TestSpecimenRewritePreservesAuthoredStrings(t *testing.T) {
	const source = `package specimen
import "example.org/specimen/helper"
var _ = helper.Value
func Design() {
 Meta("struct:field:type", "custom.T", "example.org/specimen/custom")
 Meta("struct:pkg:path", "example.org/specimen/types")
 Example("example.org/specimen/authored")
 Enum("example.org/specimen/enum")
 Default("example.org/specimen/default")
}
`
	got, err := rewriteSpecimenImports([]byte(source), "example.org/specimen", "example.com/probe/specimen")
	require.NoError(t, err)
	require.Contains(t, string(got), `import "example.com/probe/specimen/helper"`)
	require.Contains(t, string(got), `Meta("struct:field:type", "custom.T", "example.com/probe/specimen/custom")`)
	require.Contains(t, string(got), `Meta("struct:pkg:path", "example.com/probe/specimen/types")`)
	require.Contains(t, string(got), `Example("example.org/specimen/authored")`)
	require.Contains(t, string(got), `Enum("example.org/specimen/enum")`)
	require.Contains(t, string(got), `Default("example.org/specimen/default")`)
}
