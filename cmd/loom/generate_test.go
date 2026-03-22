package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeGenerator struct {
	writeErr   error
	compileErr error
	runErr     error
	runFiles   []string
	removed    bool
	writeDebug bool
	compileDbg bool
	runDebug   bool
}

func (f *fakeGenerator) Write(debug bool) error {
	f.writeDebug = debug
	return f.writeErr
}

func (f *fakeGenerator) Compile(debug bool) error {
	f.compileDbg = debug
	return f.compileErr
}

func (f *fakeGenerator) Run(debug bool) ([]string, error) {
	f.runDebug = debug
	return f.runFiles, f.runErr
}

func (f *fakeGenerator) Remove() {
	f.removed = true
}

func TestGenerateRemovesTempDirOnSuccessWithoutDebug(t *testing.T) {
	fake := &fakeGenerator{runFiles: []string{"gen/service.go"}}
	origNewGenerator := newGenerator
	newGenerator = func(cmd, path, output string, debug bool) generatorRunner {
		require.Equal(t, "gen", cmd)
		require.Equal(t, ".", output)
		require.False(t, debug)
		return fake
	}
	defer func() { newGenerator = origNewGenerator }()

	err := generate("gen", "archive/tar", ".", false)
	require.NoError(t, err)
	require.True(t, fake.removed)
	require.False(t, fake.writeDebug)
	require.False(t, fake.compileDbg)
	require.False(t, fake.runDebug)
}

func TestGenerateKeepsTempDirInDebugMode(t *testing.T) {
	fake := &fakeGenerator{runFiles: []string{"gen/service.go"}}
	origNewGenerator := newGenerator
	newGenerator = func(cmd, path, output string, debug bool) generatorRunner {
		require.True(t, debug)
		return fake
	}
	defer func() { newGenerator = origNewGenerator }()

	err := generate("gen", "archive/tar", ".", true)
	require.NoError(t, err)
	require.False(t, fake.removed)
	require.True(t, fake.writeDebug)
	require.True(t, fake.compileDbg)
	require.True(t, fake.runDebug)
}

func TestGenerateRemovesTempDirOnCompileFailureWithoutDebug(t *testing.T) {
	fake := &fakeGenerator{compileErr: errors.New("compile failed")}
	origNewGenerator := newGenerator
	newGenerator = func(cmd, path, output string, debug bool) generatorRunner {
		return fake
	}
	defer func() { newGenerator = origNewGenerator }()

	err := generate("gen", "archive/tar", ".", false)
	require.EqualError(t, err, "compile failed")
	require.True(t, fake.removed)
}

func TestCleanupDirsReturnsSubdirectoriesOnly(t *testing.T) {
	root := t.TempDir()
	genDir := filepath.Join(root, "gen")
	require.NoError(t, os.MkdirAll(filepath.Join(genDir, "svc"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(genDir, "views"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(genDir, "README.txt"), []byte("ignore"), 0o644))

	dirs := cleanupDirs("gen", root)
	require.ElementsMatch(t, []string{
		filepath.Join(genDir, "svc"),
		filepath.Join(genDir, "views"),
	}, dirs)
}

func TestCleanupDirsFallsBackWhenReaddirFails(t *testing.T) {
	root := t.TempDir()
	genDir := filepath.Join(root, "gen")
	require.NoError(t, os.WriteFile(genDir, []byte("not a dir"), 0o644))

	dirs := cleanupDirs("gen", root)
	require.Equal(t, []string{genDir}, dirs)
}

func TestCleanupDirsIgnoresNonGenCommands(t *testing.T) {
	require.Nil(t, cleanupDirs("example", t.TempDir()))
}

func TestHelpIncludesCommandsAndFlags(t *testing.T) {
	origStderr := os.Stderr
	tmp, err := os.CreateTemp(t.TempDir(), "loom-help-*.txt")
	require.NoError(t, err)
	os.Stderr = tmp
	defer func() {
		os.Stderr = origStderr
	}()

	help()

	require.NoError(t, tmp.Close())
	out, err := os.ReadFile(tmp.Name())
	require.NoError(t, err)
	text := string(out)
	require.Contains(t, text, "loom gen PACKAGE")
	require.Contains(t, text, "loom example PACKAGE")
	require.Contains(t, text, "-output DIRECTORY")
	require.Contains(t, text, "-debug")
	require.True(t, strings.Contains(text, "Loom framework") || strings.Contains(text, "Loom"))
}
