package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
)

// TestGenerationRecoveryKeepsOriginals checks recovery ownership after failed restoration.
func TestGenerationRecoveryKeepsOriginals(t *testing.T) {
	for _, kind := range []string{"external", "tree"} {
		t.Run(kind, func(t *testing.T) {
			if kind == "tree" && (runtime.GOOS == "windows" || os.Geteuid() == 0) {
				t.Skip("requires ordinary POSIX directory permissions")
			}
			dir := t.TempDir()
			tx, err := newGenerationTransaction(dir)
			require.NoError(t, err)
			destination := filepath.Join(dir, "owned.txt")
			suffix := ""
			var rollbackErr error
			if kind == "external" {
				require.NoError(t, os.WriteFile(destination, []byte("original"), 0600))
				staged := filepath.Join(tx.stage, "owned.txt")
				require.NoError(t, os.WriteFile(staged, []byte("generated"), 0600))
				outputs, err := tx.prepareExternalOutputs([]codegen.Artifact{{Path: staged}})
				require.NoError(t, err)
				require.NoError(t, tx.installExternalOutputs(outputs))
				require.NoError(t, os.Remove(destination))
				require.NoError(t, os.Mkdir(destination, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(destination, "blocker"), []byte("concurrent owner"), 0600))
				rollbackErr = tx.rollbackInstalledOutputs(outputs, filepath.Join(dir, "gen"))
			} else {
				destination = filepath.Join(dir, "gen")
				suffix = "original.txt"
				require.NoError(t, os.Mkdir(destination, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(destination, suffix), []byte("original"), 0600))
				stagedGen := filepath.Join(tx.stage, "gen")
				require.NoError(t, os.Mkdir(stagedGen, 0700))
				require.NoError(t, os.Mkdir(filepath.Join(stagedGen, "blocked"), 0700))
				require.NoError(t, os.WriteFile(filepath.Join(stagedGen, "blocked", "value.txt"), []byte("generated"), 0600))
				require.NoError(t, os.Chmod(filepath.Join(stagedGen, "blocked"), 0000))
				t.Cleanup(func() {
					for _, parent := range []string{stagedGen, destination} {
						err := os.Chmod(filepath.Join(parent, "blocked"), 0700)
						if !errors.Is(err, os.ErrNotExist) {
							require.NoError(t, err)
						}
					}
				})
				staged := filepath.Join(tx.stage, "fail.txt")
				require.NoError(t, os.WriteFile(staged, []byte("prepared bytes"), 0600))
				outputs, err := tx.prepareExternalOutputs([]codegen.Artifact{{Path: staged}})
				require.NoError(t, err)
				installRecoveryTree(t, tx)
				require.NoError(t, os.Remove(outputs[0].staged))
				installErr := tx.installExternalOutputs(outputs)
				require.ErrorContains(t, installErr, "replace generated output")
				rollbackErr = errors.Join(installErr, tx.rollbackInstalledOutputs(outputs, destination))
				require.ErrorContains(t, rollbackErr, "restore original", "fixture must actually deny tree removal")
			}
			require.Error(t, rollbackErr)
			backups, err := filepath.Glob(filepath.Join(dir, ".loom-gen-backup-*"))
			require.NoError(t, err)
			require.Len(t, backups, 1)
			backup := backups[0]
			originalPath := filepath.Join(backup, suffix)
			before, err := os.ReadFile(originalPath)
			require.NoError(t, err)
			require.Equal(t, "original", string(before))
			for range 2 {
				cleanupErr := tx.cleanup()
				assert.Error(t, cleanupErr, "unresolved recovery must remain visible")
				joined := errors.Join(rollbackErr, cleanupErr)
				assert.Contains(t, joined.Error(), backup)
				assert.Contains(t, joined.Error(), destination)
				after, err := os.ReadFile(originalPath)
				assert.NoError(t, err, "cleanup must retain the only recoverable original")
				assert.Equal(t, before, after)
			}
		})
	}
}

// TestGenerationRecoveryNormalCompletion preserves normal commit and rollback cleanup.
func TestGenerationRecoveryNormalCompletion(t *testing.T) {
	for _, fail := range []bool{false, true} {
		name := "commit"
		if fail {
			name = "rollback"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			tx, err := newGenerationTransaction(dir)
			require.NoError(t, err)
			for _, root := range []string{dir, tx.stage} {
				require.NoError(t, os.Mkdir(filepath.Join(root, "gen"), 0700))
				value := "original"
				if root == tx.stage {
					value = "generated"
				}
				require.NoError(t, os.WriteFile(filepath.Join(root, "gen", "value.txt"), []byte(value), 0600))
				require.NoError(t, os.WriteFile(filepath.Join(root, "owned.txt"), []byte(value), 0600))
			}
			outputs := []codegen.Artifact{{Path: filepath.Join(tx.stage, "gen", "value.txt")}, {Path: filepath.Join(tx.stage, "owned.txt")}}
			if fail {
				staged := filepath.Join(tx.stage, "fail.txt")
				require.NoError(t, os.WriteFile(staged, []byte("prepared bytes"), 0600))
				outputs = append(outputs, codegen.Artifact{Path: staged})
				external, prepareErr := tx.prepareExternalOutputs(outputs)
				require.NoError(t, prepareErr)
				require.Len(t, external, 2)
				installRecoveryTree(t, tx)
				require.NoError(t, os.Remove(external[1].staged))
				err = tx.installExternalOutputs(external)
				require.ErrorContains(t, err, "replace generated output")
				require.True(t, external[0].installed, "failure must follow the first external installation")
				require.NoError(t, tx.rollbackInstalledOutputs(external, filepath.Join(dir, "gen")))
			} else {
				_, err = tx.commit(outputs)
			}
			expected := "generated"
			if fail {
				require.Error(t, err)
				expected = "original"
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, tx.cleanup())
			require.NoError(t, tx.cleanup())
			for _, file := range []string{"owned.txt", "gen/value.txt"} {
				content, err := os.ReadFile(filepath.Join(dir, file))
				require.NoError(t, err)
				require.Equal(t, expected, string(content))
			}
			backups, err := filepath.Glob(filepath.Join(dir, ".loom-gen-backup-*"))
			require.NoError(t, err)
			require.Empty(t, backups)
		})
	}
}

// installRecoveryTree drives the existing commit seam before external publication.
func installRecoveryTree(t *testing.T, tx *generationTransaction) {
	t.Helper()
	live := filepath.Join(tx.output, "gen")
	backup, err := tx.preserveOriginal(live)
	require.NoError(t, err)
	require.NotNil(t, backup)
	tx.genBackup = backup
	require.DirExists(t, backup.path, "the original tree must already be preserved")
	require.NoError(t, os.Rename(filepath.Join(tx.stage, "gen"), live))
	require.DirExists(t, live, "the generated tree must already be installed")
}
