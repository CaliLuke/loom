package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
)

func TestArtifactWriteIntentRejectsLiveAliases(t *testing.T) {
	for _, test := range []struct {
		name   string
		target string
		suffix string
	}{
		{name: "existing", target: "gen", suffix: "custom.txt"},
		{name: "absent-descendants", target: "gen", suffix: "missing/nested/custom.txt"},
		{name: "dangling-alias", target: "gen/missing", suffix: "nested/custom.txt"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			live := filepath.Join(dir, "gen")
			require.NoError(t, os.Mkdir(live, 0o750))
			userFile := filepath.Join(live, "custom.txt")
			require.NoError(t, os.WriteFile(userFile, []byte("user"), 0o600))
			before, err := os.Stat(live)
			require.NoError(t, err)
			require.NoError(t, os.Symlink(test.target, filepath.Join(dir, "alias")))
			tx, err := newGenerationTransaction(dir)
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, tx.cleanup())
			})
			require.NoError(t, os.MkdirAll(filepath.Join(tx.stage, "gen"), 0o750))
			require.NoError(t, writeStagedManifest(tx.stage))
			file := &codegen.File{
				Path:      filepath.Join("alias", test.suffix),
				SkipExist: true,
				Sections:  []codegen.Section{codegen.NewRawSection("content", "template")},
			}
			staged, err := file.Render(tx.stage)
			require.NoError(t, err)
			_, err = finishGenerationTransaction(tx, []codegen.Artifact{{Path: staged, SkipExist: true}}, false)
			require.ErrorContains(t, err, "must be outside gen/")
			got, err := os.ReadFile(userFile)
			require.NoError(t, err)
			require.Equal(t, "user", string(got))
			after, err := os.Stat(live)
			require.NoError(t, err)
			require.True(t, os.SameFile(before, after))
			require.NoDirExists(t, filepath.Join(live, "missing"))
		})
	}
}

func TestArtifactWriteIntentRejectsAliasBeforePreparingBatch(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "gen"), 0o750))
	user := filepath.Join(dir, "gen", "custom.txt")
	require.NoError(t, os.WriteFile(user, []byte("user"), 0o600))
	require.NoError(t, os.Symlink("gen", filepath.Join(dir, "alias")))
	tx, err := newGenerationTransaction(dir)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, tx.cleanup())
	})
	var artifacts []codegen.Artifact
	for _, path := range []string{"new-dir/new.txt", "alias/custom.txt"} {
		file := &codegen.File{
			Path:      path,
			SkipExist: true,
			Sections:  []codegen.Section{codegen.NewRawSection("content", "template")},
		}
		staged, err := file.Render(tx.stage)
		require.NoError(t, err)
		artifacts = append(artifacts, codegen.Artifact{Path: staged, SkipExist: true})
	}
	_, err = tx.commit(artifacts)
	require.ErrorContains(t, err, "must be outside gen/")
	require.NoDirExists(t, filepath.Join(dir, "new-dir"))
	got, err := os.ReadFile(user)
	require.NoError(t, err)
	require.Equal(t, "user", string(got))
}
