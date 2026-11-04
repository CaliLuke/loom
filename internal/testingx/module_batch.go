package testingx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	// ModuleBatch shares dependency resolution, vet and race-test invocations
	// across generated contracts with the same Loom source and dependencies.
	// Each case retains a separate package namespace and test binary.
	ModuleBatch struct {
		dir, source string
		export      bool
		cases       []*ModuleCase
	}

	// ModuleCase owns the package namespace and handwritten harness of a case.
	ModuleCase struct {
		// Dir is the directory where the caller renders generated files.
		Dir string
		// ModulePath is the import prefix to pass to the generator for this case.
		ModulePath string
		original   string
	}
)

// NewModuleBatch creates one temporary module using the already-resolved Loom
// source. A nonempty exportDir preserves standalone modules at stable case
// paths for opt-in artifact comparisons; Check then validates each separately.
func NewModuleBatch(t *testing.T, source, exportDir string) *ModuleBatch {
	t.Helper()
	batch := &ModuleBatch{dir: exportDir, source: source, export: exportDir != ""}
	if !batch.export {
		batch.dir = t.TempDir()
		batch.writeModule(t, batch.dir, "example.com/contracts")
	}
	return batch
}

// Case allocates a unique case directory. Names are relative paths; spaces,
// hyphens and equals signs are normalized for use in Go import paths. Exported
// cases retain their original names and module paths for byte comparisons.
// Add cases sequentially and finish their subtests before calling Check.
func (b *ModuleBatch) Case(t *testing.T, name, originalModule string) *ModuleCase {
	t.Helper()
	require.True(t, filepath.IsLocal(name), "case name must be local: %q", name)
	modulePath := originalModule
	if !b.export {
		name = strings.NewReplacer(" ", "_", "-", "_", "=", "_").Replace(name)
		modulePath = "example.com/contracts/" + filepath.ToSlash(name)
	}
	dir := filepath.Join(b.dir, name)
	for _, previous := range b.cases {
		require.NotEqual(t, previous.Dir, dir, "duplicate contract case %s", name)
	}
	require.NoError(t, os.MkdirAll(dir, 0o700))
	if b.export {
		b.writeModule(t, dir, modulePath)
	}
	c := &ModuleCase{Dir: dir, ModulePath: modulePath, original: originalModule}
	b.cases = append(b.cases, c)
	return c
}

// WriteFile writes a handwritten fixture, rebasing its module imports into the
// case namespace. Generated files must instead be rendered with ModulePath.
func (c *ModuleCase) WriteFile(t *testing.T, name, source string) {
	t.Helper()
	require.True(t, filepath.IsLocal(name), "fixture name must be local: %q", name)
	path := filepath.Join(c.Dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	source = strings.ReplaceAll(source, c.original+"/", c.ModulePath+"/")
	require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
}

// Check tidies, vets and race-tests every generated package and runtime harness.
// Failed setup cannot turn a partial batch into a passing result. The bounded
// test timeout applies to each test binary, not compilation of the batch.
// A -run filter may leave a batch empty; no commands run in that case.
// Explicit vet runs all analyzers, so the subsequent test does not repeat vet.
func (b *ModuleBatch) Check(t *testing.T) {
	t.Helper()
	if t.Failed() || len(b.cases) == 0 {
		return
	}
	dirs := []string{b.dir}
	if b.export {
		dirs = nil
		for _, c := range b.cases {
			dirs = append(dirs, c.Dir)
		}
	}
	for _, dir := range dirs {
		for _, args := range [][]string{{"mod", "tidy"}, {"vet", "./..."}, {"test", "-vet=off", "-race", "-count=1", "-timeout=30s", "-v", "./..."}} {
			output, err := RunCmd(dir, "go", args...)
			require.NoError(t, err, output)
			t.Log(output)
		}
	}
}

func (b *ModuleBatch) writeModule(t *testing.T, dir, module string) {
	t.Helper()
	content := fmt.Sprintf("module %s\n\ngo 1.27.0\n\nrequire github.com/CaliLuke/loom v1.0.0\n\nreplace github.com/CaliLuke/loom => %q\n", module, b.source)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o600))
}
