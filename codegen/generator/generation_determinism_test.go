package generator

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	servicetestdata "github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/eval"
	grpctestdata "github.com/CaliLuke/loom/grpc/codegen/testdata"
)

const generationDeterminismHelperPath = "LOOM_GENERATION_DETERMINISM_HELPER_PATH"

// generationDeterminismDSLs lists designs whose generated imports come from
// several struct:pkg:path packages or struct:field:type meta types.
var generationDeterminismDSLs = []struct {
	name string
	dsl  func()
}{
	{"ConvertMultiPkgDSL", servicetestdata.ConvertMultiPkgDSL},
	{"PkgPathMultipleDSL", servicetestdata.PkgPathMultipleDSL},
	{"StructMetaTypeDSL", grpctestdata.StructMetaTypeDSL},
	{"PayloadWithNestedTypesDSL", grpctestdata.PayloadWithNestedTypesDSL},
}

// TestGeneratedFilesAreByteStableAcrossIndependentProcesses generates the
// service, transport, and OpenAPI files of each design in separate processes
// and requires the unformatted section output and the rendered files to be
// byte-identical. Map iteration order changes between processes, so repeated
// generation within one process cannot detect map-ordered output.
func TestGeneratedFilesAreByteStableAcrossIndependentProcesses(t *testing.T) {
	if path := os.Getenv(generationDeterminismHelperPath); path != "" {
		require.NoError(t, os.WriteFile(path, renderGenerationDeterminismDump(t), 0o600))
		return
	}

	const runs = 5
	outputs := make([]string, 0, runs)
	for range runs {
		path := filepath.Join(t.TempDir(), "dump.txt")
		cmd := exec.Command(os.Args[0], "-test.run=^TestGeneratedFilesAreByteStableAcrossIndependentProcesses$")
		cmd.Env = append(os.Environ(), generationDeterminismHelperPath+"="+path)
		output, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "isolated generation failed:\n%s", output)
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		outputs = append(outputs, string(content))
	}
	for i, content := range outputs[1:] {
		require.Equal(t, outputs[0], content, "generated output differs in isolated generation %d", i+2)
	}
}

// renderGenerationDeterminismDump returns, for every design and generated
// file, the raw section output followed by the rendered file content.
func renderGenerationDeterminismDump(t *testing.T) []byte {
	t.Helper()
	var dump bytes.Buffer
	for _, tc := range generationDeterminismDSLs {
		root := codegen.RunDSL(t, tc.dsl)
		roots := []eval.Root{root}
		var files []*codegen.File
		for _, gen := range []func(string, []eval.Root) ([]*codegen.File, error){Service, Transport, OpenAPI} {
			generated, err := gen("example.com/determinism/gen", roots)
			require.NoError(t, err, tc.name)
			files = append(files, generated...)
		}
		dir := t.TempDir()
		for _, file := range mergeFilesByPath(files) {
			fmt.Fprintf(&dump, "== %s %s (raw)\n", tc.name, filepath.ToSlash(file.Path))
			for _, section := range file.AllSections() {
				require.NoError(t, section.Write(&dump), file.Path)
			}
			rendered, err := file.Render(dir)
			require.NoError(t, err, file.Path)
			content, err := os.ReadFile(rendered)
			require.NoError(t, err, rendered)
			fmt.Fprintf(&dump, "== %s %s (rendered)\n%s", tc.name, filepath.ToSlash(file.Path), content)
		}
	}
	return dump.Bytes()
}
