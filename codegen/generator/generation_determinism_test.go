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
	httptestdata "github.com/CaliLuke/loom/http/codegen/testdata"
)

const generationDeterminismHelperPath = "LOOM_GENERATION_DETERMINISM_HELPER_PATH"

// generationDeterminismDSLs lists designs whose generated imports come from
// several packages, custom Go types, parameter serialization grammars, and
// response selection, informational metadata, and SSE handshake contracts.
var generationDeterminismDSLs = []struct {
	name string
	dsl  func()
}{
	{"ConvertMultiPkgDSL", servicetestdata.ConvertMultiPkgDSL},
	{"PkgPathMultipleDSL", servicetestdata.PkgPathMultipleDSL},
	{"StructMetaTypeDSL", grpctestdata.StructMetaTypeDSL},
	{"PayloadWithNestedTypesDSL", grpctestdata.PayloadWithNestedTypesDSL},
	{"OptionalCollectionValidationDSL", optionalCollectionValidationDSL},
	{"UntaggedArraysDSL", httptestdata.UntaggedArraysDSL},
	{"UntaggedRootsDSL", httptestdata.UntaggedRootsDSL},
	{"UntaggedLocationsDSL", httptestdata.UntaggedLocationsDSL},
	{"UntaggedNullFieldsDSL", httptestdata.UntaggedNullFieldsDSL},
	{"UntaggedNullFieldsJSONRPCDSL", httptestdata.UntaggedNullFieldsJSONRPCDSL},
	{"GRPCUntaggedNullFieldsDSL", grpctestdata.UntaggedNullFieldsDSL},
	{"AdditionalCompositeDSL", httptestdata.AdditionalCompositeDSL},
	{"RequiredOccurrenceDSL", httptestdata.RequiredOccurrenceDSL},
	{"AdditionalCompositeJSONRPCDSL", httptestdata.AdditionalCompositeJSONRPCDSL},
	{"PayloadQueryStringDSL", httptestdata.PayloadQueryStringDSL},
	{"PayloadQueryArrayStringValidateDSL", httptestdata.PayloadQueryArrayStringValidateDSL},
	{"PayloadQueryMapStringStringDSL", httptestdata.PayloadQueryMapStringStringDSL},
	{"PayloadMapQueryPrimitivePrimitiveDSL", httptestdata.PayloadMapQueryPrimitivePrimitiveDSL},
	{"PayloadPathArrayStringDSL", httptestdata.PayloadPathArrayStringDSL},
	{"PayloadHeaderArrayStringDSL", httptestdata.PayloadHeaderArrayStringDSL},
	{"PayloadCookieStringDSL", httptestdata.PayloadCookieStringDSL},
	{"PayloadFormBodyObjectDSL", httptestdata.PayloadFormBodyObjectDSL},
	{"MappedNamesDSL", httptestdata.MappedNamesDSL},
	{"SecurityDSL", httptestdata.SecurityDSL},
	{"RawRequestBodyOpenAPIDSL", httptestdata.RawRequestBodyOpenAPIDSL},
	{"OpenAPI32FeaturesDSL", httptestdata.OpenAPI32FeaturesDSL},
	{"ParameterStylesDSL", httptestdata.ParameterStylesDSL},
	{"ParameterStyles31DSL", httptestdata.ParameterStyles31DSL},
	{"StyledMediaContractsDSL", httptestdata.StyledMediaContractsDSL},
	{"ParameterPropertyNamesDSL", httptestdata.ParameterPropertyNamesDSL},
	{"ParameterMetadataDSL", httptestdata.ParameterMetadataDSL},
	{"ResponseStatusSelectionDSL", httptestdata.ResponseStatusSelectionDSL},
	{"InformationalServerDSL", httptestdata.InformationalServerDSL},
	{"SSEMultiExactHandshakeDSL", func() {
		httptestdata.ConstructorOwnership("metadata-multi-exact", false, false)
	}},
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

// TestGenerationDeterminismHandshakeFixture checks the new multi-exact variant
// and preserves the singleton inputs already used by generated wire tests.
func TestGenerationDeterminismHandshakeFixture(t *testing.T) {
	for _, tc := range []struct {
		mode     string
		statuses []int
	}{
		{"metadata", []int{201}},
		{"metadata-handshake", []int{201}},
		{"metadata-multi-exact", []int{201, 202}},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				httptestdata.ConstructorOwnership(tc.mode, false, false)
			})
			require.Len(t, root.API.HTTP.Services, 1)
			endpoint := root.API.HTTP.Services[0].Endpoint("watch")
			require.NotNil(t, endpoint)
			require.NotNil(t, endpoint.SSE)
			require.Len(t, endpoint.Responses, len(tc.statuses))
			for i, response := range endpoint.Responses {
				status, exact := response.Status.Exact()
				require.True(t, exact)
				require.Equal(t, tc.statuses[i], status)
				require.Equal(t, "status", response.StatusField)
				require.NotNil(t, response.Handshake)
				require.True(t, response.Handshake.IsRequired("status"))
				require.Equal(t, endpoint.Responses[0].Handshake.Type, response.Handshake.Type)
			}
		})
	}
}
