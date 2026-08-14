package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	loom "github.com/CaliLuke/loom/pkg"
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
	output     string
	runFunc    func(string) error
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
	if f.runFunc != nil {
		if err := f.runFunc(f.output); err != nil {
			return nil, err
		}
	} else if f.output != "" && f.runErr == nil {
		if err := os.MkdirAll(filepath.Join(f.output, "gen"), 0o755); err != nil {
			return nil, err
		}
		if err := writeStagedManifest(f.output); err != nil {
			return nil, err
		}
		for _, output := range f.runFiles {
			if filepath.IsAbs(output) || pathEscapes(output) {
				continue
			}
			path := filepath.Join(f.output, output)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(path, nil, 0o644); err != nil {
				return nil, err
			}
		}
	}
	return f.runFiles, f.runErr
}

func (f *fakeGenerator) Remove() error {
	f.removed = true
	return nil
}

func TestGenerateRemovesTempDirOnSuccessWithoutDebug(t *testing.T) {
	t.Chdir(t.TempDir())
	fake := &fakeGenerator{runFiles: []string{"gen/service.go"}}
	origNewGenerator := newGenerator
	newGenerator = func(cmd, path, output string, debug bool) generatorRunner {
		require.Equal(t, "gen", cmd)
		require.Equal(t, filepath.Dir(currentDirectory()), filepath.Dir(output))
		require.Contains(t, filepath.Base(output), ".loom-gen-")
		require.False(t, debug)
		fake.output = output
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
	t.Chdir(t.TempDir())
	fake := &fakeGenerator{runFiles: []string{"gen/service.go"}}
	origNewGenerator := newGenerator
	newGenerator = func(cmd, path, output string, debug bool) generatorRunner {
		require.True(t, debug)
		fake.output = output
		return fake
	}
	defer func() { newGenerator = origNewGenerator }()

	err := generate("gen", "archive/tar", ".", true)
	require.NoError(t, err)
	require.False(t, fake.removed)
	require.True(t, fake.writeDebug)
	require.True(t, fake.compileDbg)
	require.True(t, fake.runDebug)
	requireNoGenerationArtifacts(t, filepath.Dir(currentDirectory()))
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
				"stage=Validate",
				"stage=Commit",
				"stage=total",
			},
			wantKeepTmpDir: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			fake := &fakeGenerator{runFiles: []string{"gen/service.go", "gen/client.go"}}
			origNewGenerator := newGenerator
			newGenerator = func(cmd, path, output string, debug bool) generatorRunner {
				require.Equal(t, tc.debug, debug)
				fake.output = output
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
		fake.output = output
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
				tc.fake.output = output
				return tc.fake
			}
			defer func() { newGenerator = origNewGenerator }()

			err := generate("gen", "archive/tar", ".", false)
			require.EqualError(t, err, tc.wantErr)
		})
	}
}

func TestGenerateTransactionReplacesGenOnlyAfterSuccess(t *testing.T) {
	output := t.TempDir()
	liveGen := filepath.Join(output, "gen")
	require.NoError(t, os.MkdirAll(liveGen, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(liveGen, "stale.go"), []byte("stale"), 0o644))

	fake := &fakeGenerator{
		runFiles: []string{"gen/current.go"},
		runFunc: func(stage string) error {
			require.FileExists(t, filepath.Join(liveGen, "stale.go"))
			gen := filepath.Join(stage, "gen")
			if err := os.MkdirAll(gen, 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(gen, "current.go"), []byte("current"), 0o644); err != nil {
				return err
			}
			return writeStagedManifest(stage)
		},
	}
	stubGeneratorFactory(t, fake)

	stdout, _, err := captureOutput(t, func() error {
		return generate("gen", "archive/tar", output, false)
	})

	require.NoError(t, err)
	expectedOutput, err := filepath.Rel(currentDirectory(), filepath.Join(output, "gen", "current.go"))
	require.NoError(t, err)
	require.Equal(t, expectedOutput+"\n", stdout)
	require.NoFileExists(t, filepath.Join(liveGen, "stale.go"))
	require.FileExists(t, filepath.Join(liveGen, "current.go"))
	requireNoGenerationArtifacts(t, filepath.Dir(output))
}

func TestGenerateTransactionCommitsExternalPluginOutputs(t *testing.T) {
	output := t.TempDir()
	liveGen := filepath.Join(output, "gen")
	external := filepath.Join(output, "AGENTS_QUICKSTART.md")
	require.NoError(t, os.MkdirAll(liveGen, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(liveGen, "stale.go"), []byte("stale"), 0o644))
	require.NoError(t, os.WriteFile(external, []byte("old guide"), 0o644))

	fake := &fakeGenerator{
		runFiles: []string{"gen/current.go", "AGENTS_QUICKSTART.md"},
		runFunc: func(stage string) error {
			gen := filepath.Join(stage, "gen")
			if err := os.MkdirAll(gen, 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(gen, "current.go"), []byte("current"), 0o644); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(stage, "AGENTS_QUICKSTART.md"), []byte("new guide"), 0o644); err != nil {
				return err
			}
			return writeStagedManifest(stage)
		},
	}
	stubGeneratorFactory(t, fake)

	_, _, err := captureOutput(t, func() error {
		return generate("gen", "archive/tar", output, false)
	})

	require.NoError(t, err)
	require.NoFileExists(t, filepath.Join(liveGen, "stale.go"))
	require.FileExists(t, filepath.Join(liveGen, "current.go"))
	contents, err := os.ReadFile(external)
	require.NoError(t, err)
	require.Equal(t, "new guide", string(contents))
	requireNoGenerationArtifacts(t, filepath.Dir(output))
}

func TestGenerateTransactionRollsBackExternalPluginOutputs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("read-only directory permissions differ on Windows")
	}

	output := t.TempDir()
	liveGen := filepath.Join(output, "gen")
	firstExternal := filepath.Join(output, "FIRST.md")
	lockedDir := filepath.Join(output, "locked")
	secondExternal := filepath.Join(lockedDir, "SECOND.md")
	require.NoError(t, os.MkdirAll(liveGen, 0o755))
	require.NoError(t, os.MkdirAll(lockedDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(liveGen, "original.go"), []byte("original"), 0o644))
	require.NoError(t, os.WriteFile(firstExternal, []byte("old first"), 0o644))
	require.NoError(t, os.WriteFile(secondExternal, []byte("old second"), 0o644))
	require.NoError(t, os.Chmod(lockedDir, 0o500))
	t.Cleanup(func() {
		require.NoError(t, os.Chmod(lockedDir, 0o755))
	})

	fake := &fakeGenerator{runFiles: []string{"gen/current.go", "FIRST.md", "locked/SECOND.md"}}
	stubGeneratorFactory(t, fake)

	err := generate("gen", "archive/tar", output, false)

	require.ErrorContains(t, err, "stage Commit")
	require.FileExists(t, filepath.Join(liveGen, "original.go"))
	require.NoFileExists(t, filepath.Join(liveGen, "current.go"))
	firstContents, readErr := os.ReadFile(firstExternal)
	require.NoError(t, readErr)
	require.Equal(t, "old first", string(firstContents))
	secondContents, readErr := os.ReadFile(secondExternal)
	require.NoError(t, readErr)
	require.Equal(t, "old second", string(secondContents))
	requireNoGenerationArtifacts(t, filepath.Dir(output))
}

func TestGenerateTransactionPreservesGenOnRunFailure(t *testing.T) {
	output := t.TempDir()
	liveGen := filepath.Join(output, "gen")
	require.NoError(t, os.MkdirAll(liveGen, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(liveGen, "service.go"), []byte("original"), 0o644))

	fake := &fakeGenerator{
		runFunc: func(stage string) error {
			require.FileExists(t, filepath.Join(liveGen, "service.go"))
			gen := filepath.Join(stage, "gen")
			if err := os.MkdirAll(gen, 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(gen, "partial.go"), []byte("partial"), 0o644); err != nil {
				return err
			}
			return errors.New("finalize failed")
		},
	}
	stubGeneratorFactory(t, fake)

	err := generate("gen", "archive/tar", output, false)

	require.EqualError(t, err, "stage Run: finalize failed")
	require.FileExists(t, filepath.Join(liveGen, "service.go"))
	require.NoFileExists(t, filepath.Join(liveGen, "partial.go"))
	requireNoGenerationArtifacts(t, filepath.Dir(output))
}

func TestGenerateTransactionPreservesGenOnStagingValidationFailure(t *testing.T) {
	output := t.TempDir()
	liveGen := filepath.Join(output, "gen")
	require.NoError(t, os.MkdirAll(liveGen, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(liveGen, "service.go"), []byte("original"), 0o644))

	fake := &fakeGenerator{runFunc: func(string) error { return nil }}
	stubGeneratorFactory(t, fake)

	err := generate("gen", "archive/tar", output, false)

	require.ErrorContains(t, err, "stage Validate")
	require.FileExists(t, filepath.Join(liveGen, "service.go"))
	requireNoGenerationArtifacts(t, filepath.Dir(output))
}

func TestGenerateTransactionRejectsMissingStagedOutput(t *testing.T) {
	output := t.TempDir()
	liveGen := filepath.Join(output, "gen")
	require.NoError(t, os.MkdirAll(liveGen, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(liveGen, "service.go"), []byte("original"), 0o644))

	fake := &fakeGenerator{
		runFiles: []string{"gen/missing.go"},
		runFunc: func(stage string) error {
			if err := os.MkdirAll(filepath.Join(stage, "gen"), 0o755); err != nil {
				return err
			}
			return writeStagedManifest(stage)
		},
	}
	stubGeneratorFactory(t, fake)

	err := generate("gen", "archive/tar", output, false)

	require.ErrorContains(t, err, "stage Validate")
	require.ErrorContains(t, err, "missing.go")
	require.FileExists(t, filepath.Join(liveGen, "service.go"))
	requireNoGenerationArtifacts(t, filepath.Dir(output))
}

func TestGenerateTransactionRejectsInvalidManifest(t *testing.T) {
	output := t.TempDir()
	liveGen := filepath.Join(output, "gen")
	require.NoError(t, os.MkdirAll(liveGen, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(liveGen, "service.go"), []byte("original"), 0o644))

	fake := &fakeGenerator{
		runFunc: func(stage string) error {
			gen := filepath.Join(stage, "gen")
			if err := os.MkdirAll(gen, 0o755); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(gen, "loom.json"), []byte(`{"loom_version":"wrong"}`), 0o644)
		},
	}
	stubGeneratorFactory(t, fake)

	err := generate("gen", "archive/tar", output, false)

	require.ErrorContains(t, err, "stage Validate")
	require.ErrorContains(t, err, "has Loom version \"wrong\"")
	require.FileExists(t, filepath.Join(liveGen, "service.go"))
	requireNoGenerationArtifacts(t, filepath.Dir(output))
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
		"stage=Validate",
		"stage=Commit",
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

func TestGenerateRealBinaryPreservesModuleQualifiedGenPackage(t *testing.T) {
	fixtureDir, err := filepath.Abs(filepath.Join("..", "..", "jsonrpc", "integration_tests", "fixtures", "ticktock"))
	require.NoError(t, err)

	origWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(fixtureDir))
	defer func() {
		require.NoError(t, os.Chdir(origWD))
	}()

	outputDir, err := os.MkdirTemp(fixtureDir, "transactional-gen-")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(outputDir))
	})

	_, _, err = captureOutput(t, func() error {
		return generate("gen", "example.com/ticktock/design", outputDir, false)
	})
	require.NoError(t, err)

	clientTypes := filepath.Join(outputDir, "gen", "jsonrpc", "clock", "client", "types.go")
	contents, err := os.ReadFile(clientTypes)
	require.NoError(t, err)
	wantImport := "example.com/ticktock/" + filepath.Base(outputDir) + "/gen/clock"
	require.Contains(t, string(contents), wantImport)

	command := exec.Command("go", "test", "./...")
	command.Dir = outputDir
	command.Env = append(os.Environ(), "GOWORK=off")
	combined, err := command.CombinedOutput()
	require.NoError(t, err, string(combined))
}

func TestGenerateRealBinaryCommitFailureCleansStage(t *testing.T) {
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
	require.ErrorContains(t, err, "stage Commit: create output directory "+outputPath)
	require.Empty(t, stdout)
	require.NotContains(t, stdout, "[loom-debug]")
	require.Contains(t, stderr, "stage=binary-startup")
	require.Contains(t, stderr, "stage=generator.Generate")
	contents, readErr := os.ReadFile(outputPath)
	require.NoError(t, readErr)
	require.Equal(t, "x", string(contents))
	requireNoGenerationArtifacts(t, outputRoot)
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
	require.Contains(t, text, "loom test-scaffold PACKAGE")
	require.Contains(t, text, "-o, --output FILE-OR-DIRECTORY (import)")
	require.Contains(t, text, "-o, -output DIRECTORY (gen, example, test-scaffold)")
	require.NotContains(t, text, "\t")
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

func stubGeneratorFactory(t *testing.T, fake *fakeGenerator) {
	t.Helper()
	origNewGenerator := newGenerator
	newGenerator = func(cmd, path, output string, debug bool) generatorRunner {
		fake.output = output
		return fake
	}
	t.Cleanup(func() {
		newGenerator = origNewGenerator
	})
}

func requireNoGenerationArtifacts(t *testing.T, dir string) {
	t.Helper()
	stages, err := filepath.Glob(filepath.Join(dir, ".loom-gen-*"))
	require.NoError(t, err)
	require.Empty(t, stages)
	backups, err := filepath.Glob(filepath.Join(dir, ".loom-gen-backup-*"))
	require.NoError(t, err)
	require.Empty(t, backups)
}

func writeStagedManifest(stage string) error {
	contents, err := json.Marshal(map[string]string{"loom_version": loom.Version()})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(stage, "gen", "loom.json"), contents, 0o644)
}
