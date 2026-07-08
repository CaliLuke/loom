package main

import (
	"bytes"
	"errors"
	"io"
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

func (f *fakeGenerator) Remove() error {
	f.removed = true
	return nil
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

func TestGeneratorWriteUsesCurrentDirectoryWhenGetwdFails(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	origGetwd := getwd
	getwd = func() (string, error) {
		return "", errors.New("getwd failed")
	}
	defer func() { getwd = origGetwd }()

	g := &Generator{
		Command:       "gen",
		DesignPath:    "archive/tar",
		DesignVersion: 1,
	}
	err := g.Write(false)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(g.tmpDir))
	}()

	tmpDir, err := filepath.Abs(g.tmpDir)
	require.NoError(t, err)
	require.Equal(t, root, filepath.Dir(tmpDir))
	require.FileExists(t, filepath.Join(tmpDir, "main.go"))
}

func TestGenerateStdoutAndStderrContract(t *testing.T) {
	cases := map[string]struct {
		debug          bool
		wantStdout     string
		wantStderr     []string
		wantNoStderr   bool
		wantKeepTmpDir bool
	}{
		"normal": {
			debug:        false,
			wantStdout:   "gen/service.go\ngen/client.go\n",
			wantNoStderr: true,
		},
		"debug": {
			debug:      true,
			wantStdout: "gen/service.go\ngen/client.go\n",
			wantStderr: []string{
				"[loom-debug]",
				"stage=build.Import",
				"stage=NewGenerator",
				"stage=Write",
				"stage=Compile",
				"stage=Run",
				"stage=total",
			},
			wantKeepTmpDir: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fake := &fakeGenerator{runFiles: []string{"gen/service.go", "gen/client.go"}}
			origNewGenerator := newGenerator
			newGenerator = func(cmd, path, output string, debug bool) generatorRunner {
				require.Equal(t, tc.debug, debug)
				return fake
			}
			defer func() { newGenerator = origNewGenerator }()

			stdout, stderr, err := captureOutput(t, func() error {
				return generate("gen", "archive/tar", ".", tc.debug)
			})

			require.NoError(t, err)
			require.Equal(t, tc.wantStdout, stdout)
			if tc.wantNoStderr {
				require.Empty(t, stderr)
			}
			for _, want := range tc.wantStderr {
				require.Contains(t, stderr, want)
			}
			require.Equal(t, tc.wantKeepTmpDir, !fake.removed)
		})
	}
}

func TestGenerateRemovesTempDirOnCompileFailureWithoutDebug(t *testing.T) {
	fake := &fakeGenerator{compileErr: errors.New("compile failed")}
	origNewGenerator := newGenerator
	newGenerator = func(cmd, path, output string, debug bool) generatorRunner {
		return fake
	}
	defer func() { newGenerator = origNewGenerator }()

	err := generate("gen", "archive/tar", ".", false)
	require.EqualError(t, err, "stage Compile: compile failed")
	require.True(t, fake.removed)
}

func TestGenerateWrapsStageFailures(t *testing.T) {
	cases := map[string]struct {
		fake    *fakeGenerator
		wantErr string
	}{
		"write": {
			fake:    &fakeGenerator{writeErr: errors.New("write failed")},
			wantErr: "stage Write: write failed",
		},
		"compile": {
			fake:    &fakeGenerator{compileErr: errors.New("compile failed")},
			wantErr: "stage Compile: compile failed",
		},
		"run": {
			fake:    &fakeGenerator{runErr: errors.New("run failed")},
			wantErr: "stage Run: run failed",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			origNewGenerator := newGenerator
			newGenerator = func(cmd, path, output string, debug bool) generatorRunner {
				return tc.fake
			}
			defer func() { newGenerator = origNewGenerator }()

			err := generate("gen", "archive/tar", ".", false)
			require.EqualError(t, err, tc.wantErr)
		})
	}
}

func TestGenerateRealBinaryDebugContract(t *testing.T) {
	fixtureDir, err := filepath.Abs(filepath.Join("..", "..", "jsonrpc", "integration_tests", "fixtures", "ticktock"))
	require.NoError(t, err)

	origWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(fixtureDir))
	defer func() {
		require.NoError(t, os.Chdir(origWD))
	}()

	outputDir := t.TempDir()
	stdout, stderr, err := captureOutput(t, func() error {
		return generate("gen", "example.com/ticktock/design", outputDir, true)
	})

	require.NoError(t, err)
	require.NotEmpty(t, stdout)
	require.NotContains(t, stdout, "[loom-debug]")
	require.NotContains(t, stdout, "[TIMING]")
	for _, want := range []string{
		"stage=build.Import",
		"stage=NewGenerator",
		"stage=design-package-load",
		"stage=Write",
		"stage=temp-package-load",
		"stage=go-build",
		"stage=Compile",
		"stage=Run",
		"stage=binary-startup",
		"stage=eval.Context.Errors",
		"stage=eval.RunDSL",
		"stage=generator.Generate",
		"stage=load-roots",
		"stage=compute-gen-package",
		"stage=load-generators",
		"stage=prepare-plugins",
		"stage=generate-initial-files",
		"stage=post-generation-plugins",
		"stage=merge-files",
		"stage=write-files",
		"stage=compute-outputs",
		"stage=total",
	} {
		require.Contains(t, stderr, want)
	}
	require.NotContains(t, stderr, "[TIMING]")
}

func TestGenerateRealBinaryFailurePreservesStageContext(t *testing.T) {
	fixtureDir, err := filepath.Abs(filepath.Join("..", "..", "jsonrpc", "integration_tests", "fixtures", "ticktock"))
	require.NoError(t, err)

	origWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(fixtureDir))
	defer func() {
		require.NoError(t, os.Chdir(origWD))
	}()

	outputRoot := t.TempDir()
	outputPath := filepath.Join(outputRoot, "not-a-dir")
	require.NoError(t, os.WriteFile(outputPath, []byte("x"), 0o644))

	stdout, stderr, err := captureOutput(t, func() error {
		return generate("gen", "example.com/ticktock/design", outputPath, true)
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "stage Run: exit status 1")
	require.ErrorContains(t, err, "stage generator.Generate: stage compute-gen-package path "+filepath.Join(outputPath, "gen"))
	require.Empty(t, stdout)
	require.NotContains(t, stdout, "[loom-debug]")
	require.Contains(t, stderr, "stage=binary-startup")
	require.Contains(t, stderr, "stage generator.Generate: stage compute-gen-package path "+filepath.Join(outputPath, "gen"))
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

func captureOutput(t *testing.T, fn func() error) (stdout string, stderr string, err error) {
	t.Helper()

	origStdout := os.Stdout
	origStderr := os.Stderr

	stdoutReader, stdoutWriter, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)
	stderrReader, stderrWriter, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)

	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	runErr := fn()

	require.NoError(t, stdoutWriter.Close())
	require.NoError(t, stderrWriter.Close())

	var stdoutBuf bytes.Buffer
	_, pipeErr = io.Copy(&stdoutBuf, stdoutReader)
	require.NoError(t, pipeErr)
	require.NoError(t, stdoutReader.Close())

	var stderrBuf bytes.Buffer
	_, pipeErr = io.Copy(&stderrBuf, stderrReader)
	require.NoError(t, pipeErr)
	require.NoError(t, stderrReader.Close())

	return stdoutBuf.String(), stderrBuf.String(), runErr
}
