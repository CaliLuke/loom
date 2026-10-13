package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
)

func TestPublicationPreparationFailurePreservesLive(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "gen")
	require.NoError(t, os.Mkdir(live, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(live, "old.txt"), []byte("original"), 0o600))
	before, err := os.Stat(live)
	require.NoError(t, err)
	tx, err := newGenerationTransaction(dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, tx.cleanup())
	})
	require.NoError(t, os.Mkdir(filepath.Join(tx.stage, "gen"), 0o750))
	first := filepath.Join(tx.stage, "first.txt")
	require.NoError(t, os.WriteFile(first, []byte("prepared"), 0o600))
	_, err = tx.commit([]codegen.Artifact{{Path: first}, {Path: filepath.Join(tx.stage, "missing.txt")}})
	require.ErrorContains(t, err, "open staged output")
	after, err := os.Stat(live)
	require.NoError(t, err)
	require.True(t, os.SameFile(before, after), "preparation must precede moving live gen")
	require.NoFileExists(t, filepath.Join(dir, "first.txt"))
	require.Len(t, tx.externalCandidates, 1)
	candidate := tx.externalCandidates[0]
	require.FileExists(t, candidate)
	require.NoError(t, tx.cleanup())
	require.NoFileExists(t, candidate)
	requireNoGenerationArtifacts(t, dir)
}

func TestPublicationPreparedContentAndMode(t *testing.T) {
	for _, createOnly := range []bool{false, true} {
		name := "replace"
		if createOnly {
			name = "create-only"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			tx, err := newGenerationTransaction(dir)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, tx.cleanup())
			})
			staged := filepath.Join(tx.stage, "script")
			require.NoError(t, os.WriteFile(staged, []byte("complete bytes"), 0o751))
			outputs, err := tx.prepareExternalOutputs([]codegen.Artifact{{Path: staged, SkipExist: createOnly}})
			require.NoError(t, err)
			require.Len(t, outputs, 1)
			require.Equal(t, dir, filepath.Dir(outputs[0].staged))
			require.NotEqual(t, staged, outputs[0].staged)
			require.NoError(t, os.WriteFile(staged, []byte("later stage edit"), 0o600))
			require.NoError(t, tx.installExternalOutputs(outputs))
			destination := filepath.Join(dir, "script")
			got, err := os.ReadFile(destination)
			require.NoError(t, err)
			require.Equal(t, "complete bytes", string(got))
			info, err := os.Stat(destination)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0o751), info.Mode().Perm())
			require.NoError(t, tx.cleanup())
			requireNoGenerationArtifacts(t, dir)
		})
	}
}

func TestPublicationExistingCreateOnlyNeedsNoParentWrite(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("requires ordinary POSIX directory permissions")
	}
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	require.NoError(t, os.Mkdir(locked, 0o750))
	destination := filepath.Join(locked, "user.txt")
	require.NoError(t, os.WriteFile(destination, []byte("user"), 0o600))
	require.NoError(t, os.Chmod(locked, 0o500))
	t.Cleanup(func() {
		require.NoError(t, os.Chmod(locked, 0o750))
	})
	tx, err := newGenerationTransaction(dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, tx.cleanup())
	})
	require.NoError(t, os.Mkdir(filepath.Join(tx.stage, "gen"), 0o750))
	require.NoError(t, os.Mkdir(filepath.Join(tx.stage, "locked"), 0o750))
	staged := filepath.Join(tx.stage, "locked", "user.txt")
	require.NoError(t, os.WriteFile(staged, []byte("generated"), 0o600))
	paths, err := tx.commit([]codegen.Artifact{{Path: staged, SkipExist: true}})
	require.NoError(t, err)
	require.Empty(t, paths)
	require.Empty(t, tx.externalCandidates)
	got, err := os.ReadFile(destination)
	require.NoError(t, err)
	require.Equal(t, "user", string(got))
}
