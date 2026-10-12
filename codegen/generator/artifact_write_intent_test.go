package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/eval"
)

func TestArtifactWriteIntentRejectsConflictingPolicies(t *testing.T) {
	for _, first := range []bool{false, true} {
		t.Run(map[bool]string{false: "replace-first", true: "create-first"}[first], func(t *testing.T) {
			t.Cleanup(func() {
				generatorLoader = generators
			})
			generatorLoader = func(string) ([]genfunc, error) {
				return []genfunc{func(string, []eval.Root) ([]*codegen.File, error) {
					return []*codegen.File{
						{
							Path:      "custom.txt",
							SkipExist: first,
							Sections:  []codegen.Section{codegen.NewRawSection("first", "first")},
						},
						{
							Path:      "nested/../custom.txt",
							SkipExist: !first,
							Sections:  []codegen.Section{codegen.NewRawSection("second", "second")},
						},
					}, nil
				}}, nil
			}
			dir := t.TempDir()
			_, err := Generate(dir, "example", false)
			require.ErrorContains(t, err, "conflicting write policies")
			_, err = os.Lstat(filepath.Join(dir, "custom.txt"))
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	}
}

func TestArtifactWriteIntentSurvivesRendering(t *testing.T) {
	t.Cleanup(func() {
		generatorLoader = generators
	})
	generatorLoader = func(string) ([]genfunc, error) {
		return []genfunc{func(string, []eval.Root) ([]*codegen.File, error) {
			return []*codegen.File{
				{
					Path:      "create.txt",
					SkipExist: true,
					Sections:  []codegen.Section{codegen.NewRawSection("content", "create")},
				},
				{Path: "replace.txt", Sections: []codegen.Section{codegen.NewRawSection("content", "replace")}},
			}, nil
		}}, nil
	}
	dir := t.TempDir()
	outputs, err := Generate(dir, "example", false)
	require.NoError(t, err)
	require.Len(t, outputs, 2)
	require.Equal(t, "create.txt", filepath.Base(outputs[0].Path))
	require.True(t, outputs[0].SkipExist)
	require.False(t, outputs[1].SkipExist)
	outputs, err = Generate(dir, "example", false)
	require.NoError(t, err)
	require.Len(t, outputs, 1)
	require.Equal(t, "replace.txt", filepath.Base(outputs[0].Path))
}

func TestArtifactWriteIntentMergesEqualPolicies(t *testing.T) {
	for _, policy := range []bool{false, true} {
		t.Run(map[bool]string{false: "replace", true: "create"}[policy], func(t *testing.T) {
			t.Cleanup(func() {
				generatorLoader = generators
			})
			generatorLoader = func(string) ([]genfunc, error) {
				return []genfunc{func(string, []eval.Root) ([]*codegen.File, error) {
					return []*codegen.File{
						{
							Path:      "merged.go",
							SkipExist: policy,
							Sections:  []codegen.Section{codegen.Header("Merged", "merged", nil), codegen.NewRawSection("one", "type One struct{}\n")},
						},
						{
							Path:      "nested/../merged.go",
							SkipExist: policy,
							Sections:  []codegen.Section{codegen.Header("Merged", "merged", nil), codegen.NewRawSection("two", "type Two struct{}\n")},
						},
					}, nil
				}}, nil
			}
			dir := t.TempDir()
			outputs, err := Generate(dir, "example", false)
			require.NoError(t, err)
			require.Len(t, outputs, 1)
			require.Equal(t, policy, outputs[0].SkipExist)
			got, err := os.ReadFile(filepath.Join(dir, "merged.go"))
			require.NoError(t, err)
			require.Contains(t, string(got), "type One struct{}")
			require.Contains(t, string(got), "type Two struct{}")
		})
	}
}

func TestArtifactWriteIntentRejectsCreateOnlyGenerationTree(t *testing.T) {
	for _, path := range []string{"gen", "gen/custom.txt", "nested/../gen/custom.txt"} {
		t.Run(path, func(t *testing.T) {
			t.Cleanup(func() {
				generatorLoader = generators
			})
			generatorLoader = func(string) ([]genfunc, error) {
				return []genfunc{func(string, []eval.Root) ([]*codegen.File, error) {
					return []*codegen.File{{Path: path, SkipExist: true, Sections: []codegen.Section{codegen.NewRawSection("content", "template")}}}, nil
				}}, nil
			}
			dir := t.TempDir()
			_, err := Generate(dir, "gen", false)
			require.ErrorContains(t, err, "create-only artifact")
			require.ErrorContains(t, err, "must be outside gen/")
			require.NoFileExists(t, filepath.Join(dir, "gen", "custom.txt"))
		})
	}
}

func TestArtifactWriteIntentRejectsLiveAliasBeforeRendering(t *testing.T) {
	t.Cleanup(func() {
		generatorLoader = generators
	})
	generatorLoader = func(string) ([]genfunc, error) {
		return []genfunc{func(string, []eval.Root) ([]*codegen.File, error) {
			return []*codegen.File{
				{
					Path:     "prepared.txt",
					Sections: []codegen.Section{codegen.NewRawSection("content", "template")},
				},
				{
					Path:      "alias/missing/custom.txt",
					SkipExist: true,
					Sections:  []codegen.Section{codegen.NewRawSection("content", "template")},
				},
			}, nil
		}}, nil
	}
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "gen"), 0o750))
	require.NoError(t, os.Symlink("gen", filepath.Join(dir, "alias")))
	_, err := Generate(dir, "example", false)
	require.ErrorContains(t, err, "must be outside gen/")
	require.NoFileExists(t, filepath.Join(dir, "prepared.txt"))
	require.NoDirExists(t, filepath.Join(dir, "gen", "missing"))
}
