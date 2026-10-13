package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
)

// TestGenerationRecoveryBeforeInstall retains originals when installation never succeeds.
func TestGenerationRecoveryBeforeInstall(t *testing.T) {
	dir := t.TempDir()
	tx, err := newGenerationTransaction(dir)
	require.NoError(t, err)
	live := filepath.Join(dir, "gen")
	require.NoError(t, os.Mkdir(live, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(live, "original.txt"), []byte("original"), 0600))
	backup, err := tx.preserveOriginal(live)
	require.NoError(t, err)
	require.NotNil(t, backup)
	require.NoError(t, os.Mkdir(live, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(live, "blocker"), []byte("other owner"), 0600))
	installErr := os.Rename(filepath.Join(tx.stage, "missing"), live)
	require.ErrorIs(t, installErr, os.ErrNotExist)
	restoreErr := backup.restore()
	require.Error(t, restoreErr)
	joined := errors.Join(installErr, restoreErr, tx.cleanup())
	require.ErrorIs(t, joined, os.ErrNotExist)
	require.ErrorContains(t, joined, backup.path)
	require.ErrorContains(t, joined, live)
	for range 2 {
		require.ErrorContains(t, tx.cleanup(), backup.path)
		content, err := os.ReadFile(filepath.Join(backup.path, "original.txt"))
		require.NoError(t, err)
		require.Equal(t, "original", string(content))
	}
	require.NoError(t, os.RemoveAll(live))
	require.NoError(t, backup.restore())
	require.NoError(t, tx.cleanup())
	require.NoDirExists(t, backup.path)
}

// TestGenerationRecoveryMixedRestore attempts all restorations despite an earlier failure.
func TestGenerationRecoveryMixedRestore(t *testing.T) {
	dir := t.TempDir()
	tx, err := newGenerationTransaction(dir)
	require.NoError(t, err)
	for _, root := range []string{dir, tx.stage} {
		value := "original"
		if root == tx.stage {
			value = "generated"
		}
		require.NoError(t, os.Mkdir(filepath.Join(root, "gen"), 0700))
		require.NoError(t, os.WriteFile(filepath.Join(root, "gen", "value.txt"), []byte(value), 0600))
		for _, name := range []string{"good.txt", "blocked.txt", "second-blocked.txt"} {
			require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(value), 0600))
		}
	}
	outputs, err := tx.prepareExternalOutputs([]codegen.Artifact{
		{Path: filepath.Join(tx.stage, "good.txt")},
		{Path: filepath.Join(tx.stage, "blocked.txt")},
		{Path: filepath.Join(tx.stage, "second-blocked.txt")},
	})
	require.NoError(t, err)
	installRecoveryTree(t, tx)
	require.NoError(t, tx.installExternalOutputs(outputs))
	for _, name := range []string{"blocked.txt", "second-blocked.txt"} {
		blocked := filepath.Join(dir, name)
		require.NoError(t, os.Remove(blocked))
		require.NoError(t, os.Mkdir(blocked, 0700))
		require.NoError(t, os.WriteFile(filepath.Join(blocked, "blocker"), []byte("other owner"), 0600))
	}
	err = tx.rollbackInstalledOutputs(outputs, filepath.Join(dir, "gen"))
	require.ErrorContains(t, err, "remove generated output")
	require.ErrorContains(t, err, "restore original")
	var pathErr *os.PathError
	require.ErrorAs(t, err, &pathErr)
	cleanupErr := tx.cleanup()
	require.ErrorContains(t, cleanupErr, outputs[1].backup.path)
	require.ErrorContains(t, cleanupErr, outputs[2].backup.path)
	for _, name := range []string{"good.txt", "gen/value.txt"} {
		value, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		require.Equal(t, "original", string(value))
	}
	for _, output := range outputs[1:] {
		require.ErrorContains(t, err, output.destination)
		value, readErr := os.ReadFile(output.backup.path)
		require.NoError(t, readErr)
		require.Equal(t, "original", string(value))
	}
	require.NoFileExists(t, outputs[0].backup.path)
	require.NoDirExists(t, tx.genBackup.path)
}

// TestGenerationCommittedBackupRemovalRetries retains cleanup responsibility after a failed discard.
func TestGenerationCommittedBackupRemovalRetries(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("requires ordinary POSIX directory permissions")
	}
	for _, tree := range []bool{false, true} {
		name := "file"
		if tree {
			name = "tree"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			tx, err := newGenerationTransaction(dir)
			require.NoError(t, err)
			live := filepath.Join(dir, "owned")
			if tree {
				require.NoError(t, os.Mkdir(live, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(live, "original"), []byte("original"), 0600))
			} else {
				require.NoError(t, os.WriteFile(live, []byte("original"), 0600))
			}
			backup, err := tx.preserveOriginal(live)
			require.NoError(t, err)
			require.NotNil(t, backup)
			require.NoError(t, os.WriteFile(live, []byte("generated"), 0600))
			require.NoError(t, os.Chmod(dir, 0500))
			t.Cleanup(func() {
				require.NoError(t, os.Chmod(dir, 0700))
			})
			require.ErrorIs(t, tx.discardOriginals(), os.ErrPermission)
			require.ErrorContains(t, tx.cleanup(), backup.path)
			require.NoError(t, os.Chmod(dir, 0700))
			require.NoError(t, tx.cleanup())
			require.NoError(t, tx.cleanup())
			_, err = os.Lstat(backup.path)
			require.ErrorIs(t, err, os.ErrNotExist)
			content, err := os.ReadFile(live)
			require.NoError(t, err)
			require.Equal(t, "generated", string(content))
		})
	}
}
