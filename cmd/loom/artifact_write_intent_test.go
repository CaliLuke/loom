package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
)

func TestArtifactWriteIntentPreservesExisting(t *testing.T) {
	output := t.TempDir()
	path := filepath.Join(output, "custom.txt")
	require.NoError(t, os.WriteFile(path, []byte("user"), 0o600))
	tx, err := newGenerationTransaction(output)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, tx.cleanup())
	})
	require.NoError(t, os.MkdirAll(filepath.Join(tx.stage, "gen"), 0o700))
	f := &codegen.File{
		Path:      "custom.txt",
		SkipExist: true,
		Sections:  []codegen.Section{codegen.NewRawSection("content", "template")},
	}
	staged, err := f.Render(tx.stage)
	require.NoError(t, err)
	_, err = tx.commit([]codegen.Artifact{{Path: staged, SkipExist: true}})
	require.NoError(t, err)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "user", string(got))
}

func artifactPaths(paths []string) []codegen.Artifact {
	artifacts := make([]codegen.Artifact, len(paths))
	for i, path := range paths {
		artifacts[i].Path = path
	}
	return artifacts
}

func TestArtifactWriteIntentLiveEntries(t *testing.T) {
	for _, kind := range []string{"absent", "file", "directory", "symlink", "dangling"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "custom.txt")
			switch kind {
			case "file":
				require.NoError(t, os.WriteFile(path, []byte("user"), 0o600))
			case "directory":
				require.NoError(t, os.Mkdir(path, 0o750))
			case "symlink":
				require.NoError(t, os.WriteFile(filepath.Join(dir, "target"), []byte("user"), 0o600))
				require.NoError(t, os.Symlink("target", path))
			case "dangling":
				require.NoError(t, os.Symlink("target", path))
			}
			before, statErr := os.Lstat(path)
			if kind == "absent" {
				require.ErrorIs(t, statErr, os.ErrNotExist)
			} else {
				require.NoError(t, statErr)
			}
			tx, err := newGenerationTransaction(dir)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, tx.cleanup())
			})
			require.NoError(t, os.MkdirAll(filepath.Join(tx.stage, "gen"), 0o750))
			staged := filepath.Join(tx.stage, "custom.txt")
			require.NoError(t, os.WriteFile(staged, []byte("template"), 0o644))
			paths, err := tx.commit([]codegen.Artifact{{Path: staged, SkipExist: true}})
			require.NoError(t, err)
			after, err := os.Lstat(path)
			require.NoError(t, err)
			if kind == "absent" {
				require.Len(t, paths, 1)
				got, err := os.ReadFile(path)
				require.NoError(t, err)
				require.Equal(t, "template", string(got))
			} else {
				require.Empty(t, paths)
				require.True(t, os.SameFile(before, after))
				require.Equal(t, before.Mode(), after.Mode())
			}
			if kind == "dangling" {
				require.NoFileExists(t, filepath.Join(dir, "target"))
			}
		})
	}
}

func TestArtifactWriteIntentPreservesCreatorAfterPreparation(t *testing.T) {
	dir := t.TempDir()
	tx, err := newGenerationTransaction(dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, tx.cleanup())
	})
	staged := filepath.Join(tx.stage, "custom.txt")
	require.NoError(t, os.WriteFile(staged, []byte("template"), 0o644))
	outputs, err := tx.prepareExternalOutputs([]codegen.Artifact{{Path: staged, SkipExist: true}})
	require.NoError(t, err)
	path := filepath.Join(dir, "custom.txt")
	require.NoError(t, os.WriteFile(path, []byte("user"), 0o600))
	require.NoError(t, tx.installExternalOutputs(outputs))
	require.False(t, outputs[0].installed)
	require.Empty(t, outputs[0].backup)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "user", string(got))
}

func TestArtifactWriteIntentRollback(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "gen"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gen", "old.txt"), []byte("old"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "protected.txt"), []byte("user"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "replace.txt"), []byte("old replace"), 0o600))
	tx, err := newGenerationTransaction(dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, tx.cleanup())
	})
	require.NoError(t, os.MkdirAll(filepath.Join(tx.stage, "gen"), 0o750))
	outputs := make([]codegen.Artifact, 0, 4)
	for _, name := range []string{"protected.txt", "created.txt", "replace.txt", "missing.txt"} {
		path := filepath.Join(tx.stage, name)
		if name != "missing.txt" {
			require.NoError(t, os.WriteFile(path, []byte("template"), 0o644))
		}
		outputs = append(outputs, codegen.Artifact{Path: path, SkipExist: name != "replace.txt"})
	}
	_, err = tx.commit(outputs)
	require.Error(t, err)
	require.NoFileExists(t, filepath.Join(dir, "created.txt"))
	require.FileExists(t, filepath.Join(dir, "gen", "old.txt"))
	for name, want := range map[string]string{"protected.txt": "user", "replace.txt": "old replace"} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		require.Equal(t, want, string(got))
	}
}

func TestArtifactWriteIntentRollbackPreservesReplacementOwner(t *testing.T) {
	dir := t.TempDir()
	tx, err := newGenerationTransaction(dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, tx.cleanup())
	})
	staged := filepath.Join(tx.stage, "custom.txt")
	require.NoError(t, os.WriteFile(staged, []byte("template"), 0o644))
	outputs, err := tx.prepareExternalOutputs([]codegen.Artifact{{Path: staged, SkipExist: true}})
	require.NoError(t, err)
	require.NoError(t, tx.installExternalOutputs(outputs))
	path := filepath.Join(dir, "custom.txt")
	require.NoError(t, os.Remove(path))
	require.NoError(t, os.WriteFile(path, []byte("user"), 0o600))
	require.NoError(t, tx.rollbackInstalledOutputs(outputs, filepath.Join(dir, "gen"), false))
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "user", string(got))
}

func TestArtifactWriteIntentRejectsMalformedHelperResult(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "gen")
	require.NoError(t, os.Mkdir(live, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(live, "old.txt"), []byte("old"), 0o600))
	fake := &fakeGenerator{runResult: []byte("gen/current.go\n")}
	stubGeneratorFactory(t, fake)
	err := generate("gen", "archive/tar", dir, false)
	require.ErrorContains(t, err, "decode generated artifacts")
	require.FileExists(t, filepath.Join(live, "old.txt"))
	requireNoGenerationArtifacts(t, filepath.Dir(dir))
}

func TestArtifactWriteIntentSkipsExistingInReadOnlyDirectory(t *testing.T) {
	dir := t.TempDir()
	tx, err := newGenerationTransaction(dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, tx.cleanup())
	})
	staged := filepath.Join(tx.stage, "custom.txt")
	require.NoError(t, os.WriteFile(staged, []byte("template"), 0o644))
	path := filepath.Join(dir, "custom.txt")
	require.NoError(t, os.WriteFile(path, []byte("user"), 0o600))
	outputs, err := tx.prepareExternalOutputs([]codegen.Artifact{{Path: staged, SkipExist: true}})
	require.NoError(t, err)
	require.NoError(t, os.Chmod(dir, 0o500))
	t.Cleanup(func() {
		require.NoError(t, os.Chmod(dir, 0o750))
	})
	require.NoError(t, tx.installExternalOutputs(outputs))
	require.False(t, outputs[0].installed)
}

func TestArtifactWriteIntentRejectsGenerationTreeBeforeCommit(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "gen")
	require.NoError(t, os.Mkdir(live, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(live, "old.txt"), []byte("old"), 0o600))
	fake := &fakeGenerator{
		runArtifacts: []codegen.Artifact{{Path: "gen/custom.txt", SkipExist: true}},
		runFunc: func(stage string) error {
			if err := os.MkdirAll(filepath.Join(stage, "gen"), 0o750); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(stage, "gen", "custom.txt"), []byte("template"), 0o644); err != nil {
				return err
			}
			return writeStagedManifest(stage)
		},
	}
	stubGeneratorFactory(t, fake)
	err := generate("gen", "archive/tar", dir, false)
	require.ErrorContains(t, err, "stage Validate: create-only artifact gen/custom.txt must be outside gen/")
	require.FileExists(t, filepath.Join(live, "old.txt"))
	require.NoFileExists(t, filepath.Join(live, "custom.txt"))
}
