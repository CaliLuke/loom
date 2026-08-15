package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/eval"
)

func TestGenerateRemovesPathsAfterAllWrites(t *testing.T) {
	t.Cleanup(func() {
		generatorLoader = generators
	})
	jsonPath := filepath.Join(codegen.Gendir, "http", "openapi.json")
	yamlPath := filepath.Join(codegen.Gendir, "http", "openapi.yaml")
	generatorLoader = func(string) ([]genfunc, error) {
		return []genfunc{
			func(string, []eval.Root) ([]*codegen.File, error) {
				return []*codegen.File{
					{
						Path:        jsonPath,
						Sections:    []codegen.Section{codegen.NewRawSection("json", "{}")},
						RemovePaths: []string{yamlPath},
					},
					{
						Path:     yamlPath,
						Sections: []codegen.Section{codegen.NewRawSection("yaml", "openapi: 3.2.0\n")},
					},
				}, nil
			},
		}, nil
	}

	dir := t.TempDir()
	outputs, err := Generate(dir, "example", false)

	require.NoError(t, err)
	require.FileExists(t, filepath.Join(dir, jsonPath))
	require.NoFileExists(t, filepath.Join(dir, yamlPath))
	for _, output := range outputs {
		require.NotEqual(t, filepath.Join(dir, yamlPath), absoluteOutputPath(t, output))
	}
}

func TestGenerateRemovesDirectoriesAfterAllWrites(t *testing.T) {
	t.Cleanup(func() {
		generatorLoader = generators
	})
	serverPath := filepath.Join(codegen.Gendir, "http", "widgets", "server", "server.go")
	clientDir := filepath.Join(codegen.Gendir, "http", "widgets", "client")
	clientPath := filepath.Join(clientDir, "client.go")
	clientTypesPath := filepath.Join(clientDir, "types.go")
	generatorLoader = func(string) ([]genfunc, error) {
		return []genfunc{
			func(string, []eval.Root) ([]*codegen.File, error) {
				return []*codegen.File{
					{
						Path:        serverPath,
						Sections:    []codegen.Section{codegen.NewRawSection("server", "package server\n")},
						RemovePaths: []string{clientDir},
					},
					{
						Path:     clientPath,
						Sections: []codegen.Section{codegen.NewRawSection("client", "package client\n")},
					},
					{
						Path:     clientTypesPath,
						Sections: []codegen.Section{codegen.NewRawSection("types", "package client\n")},
					},
				}, nil
			},
		}, nil
	}

	dir := t.TempDir()
	outputs, err := Generate(dir, "example", false)

	require.NoError(t, err)
	require.FileExists(t, filepath.Join(dir, serverPath))
	require.NoDirExists(t, filepath.Join(dir, clientDir))
	for _, output := range outputs {
		absolute := absoluteOutputPath(t, output)
		require.NotEqual(t, filepath.Join(dir, clientPath), absolute)
		require.NotEqual(t, filepath.Join(dir, clientTypesPath), absolute)
	}
}

func TestGenerateSkipsPathRemovalWhenAWriteFails(t *testing.T) {
	t.Cleanup(func() {
		generatorLoader = generators
	})
	stalePath := filepath.Join(codegen.Gendir, "http", "openapi.yaml")
	generatorLoader = func(string) ([]genfunc, error) {
		return []genfunc{
			func(string, []eval.Root) ([]*codegen.File, error) {
				return []*codegen.File{
					{
						Path:        filepath.Join(codegen.Gendir, "http", "openapi.json"),
						Sections:    []codegen.Section{codegen.NewRawSection("json", "{}")},
						RemovePaths: []string{stalePath},
					},
					{
						Path:     filepath.Join(codegen.Gendir, "broken.go"),
						Sections: []codegen.Section{codegen.NewRawSection("broken", "not Go")},
					},
				}, nil
			},
		}, nil
	}

	dir := t.TempDir()
	staleAbsolutePath := filepath.Join(dir, stalePath)
	require.NoError(t, os.MkdirAll(filepath.Dir(staleAbsolutePath), 0o750))
	require.NoError(t, os.WriteFile(staleAbsolutePath, []byte("stale"), 0o600))

	_, err := Generate(dir, "example", false)

	require.Error(t, err)
	require.FileExists(t, staleAbsolutePath)
}

func TestGenerateRejectsPathRemovalOutsideOutputDirectory(t *testing.T) {
	t.Cleanup(func() {
		generatorLoader = generators
	})
	generatorLoader = func(string) ([]genfunc, error) {
		return []genfunc{
			func(string, []eval.Root) ([]*codegen.File, error) {
				return []*codegen.File{
					{
						Path:        filepath.Join(codegen.Gendir, "http", "openapi.json"),
						Sections:    []codegen.Section{codegen.NewRawSection("json", "{}")},
						RemovePaths: []string{filepath.Join("..", "outside.txt")},
					},
				}, nil
			},
		}, nil
	}

	workspace := t.TempDir()
	dir := filepath.Join(workspace, "output")
	outsidePath := filepath.Join(workspace, "outside.txt")
	require.NoError(t, os.WriteFile(outsidePath, []byte("keep"), 0o600))

	_, err := Generate(dir, "example", false)

	require.ErrorContains(t, err, "outside output directory")
	require.FileExists(t, outsidePath)
}

func TestGenerateRejectsPathRemovalThroughSymlinkedParent(t *testing.T) {
	t.Cleanup(func() {
		generatorLoader = generators
	})
	generatorLoader = func(string) ([]genfunc, error) {
		return []genfunc{
			func(string, []eval.Root) ([]*codegen.File, error) {
				return []*codegen.File{
					{
						Path:        filepath.Join(codegen.Gendir, "http", "openapi.json"),
						Sections:    []codegen.Section{codegen.NewRawSection("json", "{}")},
						RemovePaths: []string{filepath.Join("link", "victim.txt")},
					},
				}, nil
			},
		}, nil
	}

	workspace := t.TempDir()
	dir := filepath.Join(workspace, "output")
	outsideDir := filepath.Join(workspace, "outside")
	require.NoError(t, os.MkdirAll(dir, 0o750))
	require.NoError(t, os.MkdirAll(outsideDir, 0o750))
	victimPath := filepath.Join(outsideDir, "victim.txt")
	require.NoError(t, os.WriteFile(victimPath, []byte("keep"), 0o600))
	require.NoError(t, os.Symlink(outsideDir, filepath.Join(dir, "link")))

	_, err := Generate(dir, "example", false)

	require.ErrorContains(t, err, "stage remove-files")
	require.FileExists(t, victimPath)
}

func absoluteOutputPath(t *testing.T, output string) string {
	t.Helper()
	if filepath.IsAbs(output) {
		return output
	}
	cwd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Join(cwd, output)
}
