package valuecontract

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func runProbe(t *testing.T, source revision, p probe, location, commonRoot string) probeResult {
	t.Helper()
	module := t.TempDir()
	result := probeResult{ID: p.ID}
	require.NoError(t, os.MkdirAll(location, 0700))
	moduleName, inputs := prepareProbe(t, source, p, module, location, commonRoot)
	result.Inputs = inputs
	commands := []struct {
		phase string
		args  []string
	}{
		{"gen", []string{source.Binary, "gen", moduleName + "/design"}},
		{"example", []string{source.Binary, "example", moduleName + "/design"}},
		{"tidy", []string{"go", "mod", "tidy"}},
		{"build", []string{"go", "build", "./..."}},
		{"vet", []string{"go", "vet", "./..."}},
	}
	for _, command := range commands {
		if p.Fixture != "" && command.phase == "example" {
			continue
		}
		phase := runPhase(t.Context(), module, []string{"LOOM_DIR=" + source.Source}, command.phase, command.args...)
		result.Phases = append(result.Phases, phase)
		require.NoError(t, os.WriteFile(filepath.Join(location, command.phase+".log"), []byte(phase.Output), 0600))
		if phase.Exit != 0 {
			break
		}
	}
	if firstFailure(result).Exit == 0 && p.Decoder != "" {
		installDecoder(t, module, p, commonRoot)
		phase := runPhase(t.Context(), module, []string{"LOOM_DIR=" + source.Source}, "decoder", "go", "test", "-v", "-count=1", "-run", "^TestValueContract$", ".")
		result.Phases = append(result.Phases, phase)
		require.NoError(t, os.WriteFile(filepath.Join(location, "decoder.log"), []byte(phase.Output), 0600))
		observation, err := os.ReadFile(filepath.Join(module, "observation.json"))
		if err == nil {
			result.Observation = observation
		} else {
			require.True(t, os.IsNotExist(err), "read observation: %v", err)
		}
	}
	require.NoError(t, checkInputsUnchanged(module, result.Inputs))
	var err error
	result.Artifacts, err = snapshotArtifacts(module, filepath.Join(location, "artifacts"), result.Inputs)
	require.NoError(t, err)
	if firstFailure(result).Phase != "gen" {
		require.NotEmpty(t, result.Artifacts, "successful generation must retain artifacts")
		require.Contains(t, result.Artifacts, "gen/loom.json")
	}
	writeJSON(t, filepath.Join(location, "result.json"), result)
	return result
}

func prepareProbe(t *testing.T, source revision, p probe, module, location, commonRoot string) (string, map[string]string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(module, "design"), 0700))
	if p.Fixture != "" {
		copyTree(t, filepath.Join(commonRoot, p.Fixture), module)
		inputs := captureInputs(t, module, filepath.Join(location, "inputs"))
		mod, err := os.ReadFile(filepath.Join(module, "go.mod"))
		require.NoError(t, err)
		moduleName := ""
		for _, line := range strings.Split(string(mod), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == "module" {
				moduleName = fields[1]
			}
		}
		require.NotEmpty(t, moduleName, "fixture module identity")
		updated := runPhase(t.Context(), module, nil, "fixture-source", "go", "mod", "edit", "-replace=github.com/CaliLuke/loom="+source.Source)
		require.Zero(t, updated.Exit, updated.Output)
		return moduleName, inputs
	} else {
		var design []byte
		if p.Design != "" {
			var err error
			design, err = os.ReadFile(filepath.Join(commonRoot, "internal", "valuecontract", "testdata", p.Design))
			require.NoError(t, err)
		} else {
			relative := strings.TrimPrefix(p.Import, "github.com/CaliLuke/loom/")
			require.NotEqual(t, p.Import, relative, "specimen must belong to the audited Loom checkout")
			commonImport := "example.com/valueprobe/specimen"
			require.NoError(t, copyCommonSpecimen(filepath.Join(commonRoot, filepath.FromSlash(relative)), filepath.Join(module, "specimen"), p.Import, commonImport))
			design = []byte(fmt.Sprintf(`package design

import (
 "github.com/CaliLuke/loom/eval"
 "github.com/CaliLuke/loom/expr"
 specimen %q
)

func init() {
 *expr.Root = expr.RootExpr{}
 *expr.GeneratedResultTypes = expr.ResultTypesRoot{}
 eval.Reset()
 specimen.%s()
}
`, commonImport, p.Function))
		}
		require.NoError(t, os.WriteFile(filepath.Join(module, "design", "design.go"), design, 0600))
	}
	inputs := captureInputs(t, module, filepath.Join(location, "inputs"))
	mod := fmt.Sprintf("module example.com/valueprobe\n\ngo 1.27.0\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %q\n", source.Source)
	require.NoError(t, os.WriteFile(filepath.Join(module, "go.mod"), []byte(mod), 0600))
	sum, err := os.ReadFile(filepath.Join(source.Source, "go.sum"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(module, "go.sum"), sum, 0600))
	return "example.com/valueprobe", inputs
}

func copyTree(t *testing.T, source, destination string) {
	t.Helper()
	require.NoError(t, filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			if relative == "gen" || relative == ".git" {
				return filepath.SkipDir
			}
			return os.MkdirAll(target, 0700)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0600)
	}))
}

func installDecoder(t *testing.T, module string, p probe, commonRoot string) {
	t.Helper()
	cli, err := os.ReadFile(filepath.Join(module, "gen", p.Transport, "svc", "client", "cli.go"))
	require.NoError(t, err)
	argument := "raw"
	if strings.Contains(string(cli), "svcSendBody *string") || strings.Contains(string(cli), "svcSendMessage *string") {
		argument = "&raw"
	}
	harness, err := os.ReadFile(filepath.Join(commonRoot, "internal", "valuecontract", "testdata", "decoder.go.txt"))
	require.NoError(t, err)
	code := strings.NewReplacer("TRANSPORT", p.Transport, "ARGUMENT", argument, "EXPECTED", fmt.Sprintf("%q", p.ExpectedJSON)).Replace(string(harness))
	require.NoError(t, os.WriteFile(filepath.Join(module, "contract_test.go"), []byte(code), 0600))
}

func captureInputs(t *testing.T, module, destination string) map[string]string {
	t.Helper()
	inputs := make(map[string]string)
	require.NoError(t, filepath.WalkDir(module, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(module, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		inputs[filepath.ToSlash(relative)] = digest(data)
		return os.WriteFile(target, data, 0600)
	}))
	return inputs
}

func checkInputsUnchanged(module string, inputs map[string]string) error {
	for path, expected := range inputs {
		if path == "go.mod" || path == "go.sum" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(module, filepath.FromSlash(path)))
		if err != nil {
			return fmt.Errorf("read original input %s: %w", path, err)
		}
		if digest(data) != expected {
			return fmt.Errorf("generator changed handwritten input %s", path)
		}
	}
	return nil
}
