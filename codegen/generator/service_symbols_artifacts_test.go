package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	servicedata "github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestServiceSymbolArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_SERVICE_SYMBOL_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_SERVICE_SYMBOL_OUTPUT for isolated generation comparison")
	}
	for _, tc := range serviceSymbolDesigns() {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(output, tc.name)
			require.NoError(t, os.MkdirAll(dir, 0o750))
			renderDesignModule(t, dir, "example.com/symbols", tc.design)
			files, err := OpenAPI("example.com/symbols/gen", []eval.Root{codegen.RunDSL(t, tc.design)})
			require.NoError(t, err)
			for _, file := range files {
				_, err := file.Render(dir)
				require.NoError(t, err)
			}
		})
	}
}

func TestServiceSymbolArtifactBuilds(t *testing.T) {
	output := os.Getenv("LOOM_SERVICE_SYMBOL_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_SERVICE_SYMBOL_OUTPUT to build comparison modules")
	}
	for _, tc := range serviceSymbolDesigns() {
		t.Run(tc.name, func(t *testing.T) {
			for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}} {
				output, err := testingx.RunCmd(filepath.Join(output, tc.name), "go", args...)
				require.NoError(t, err, output)
			}
		})
	}
}

func serviceSymbolDesigns() []struct {
	name   string
	design func()
} {
	return []struct {
		name   string
		design func()
	}{
		{"fixed", testdata.ServiceReservedSymbolsDSL},
		{"external", testdata.ServiceExternalSymbolsDSL},
		{"plain", func() {
			testdata.ServiceConditionalSymbolsDSL("plain")
		}},
		{"secure", func() {
			testdata.ServiceConditionalSymbolsDSL("secure")
		}},
		{"sse", func() {
			testdata.ServiceConditionalSymbolsDSL("sse")
		}},
		{"websocket", func() {
			testdata.ServiceConditionalSymbolsDSL("websocket")
		}},
		{"interceptors", func() {
			testdata.ServiceConditionalSymbolsDSL("interceptors")
		}},
		{"fixed-methods", testdata.ServiceFixedSpellingMethodsDSL},
		{"members", testdata.ServiceMemberFieldsDSL},
		{"coupling", testdata.ServiceMethodTypeCouplingDSL},
		{"carriers", testdata.ServiceCarrierSymbolsDSL},
		{"sse-event", testdata.ServiceSSEEventSymbolsDSL},
		{"ordinary", servicedata.SingleMethodDSL},
		{"external-method", servicedata.PkgPathPayloadAttributeDSL},
	}
}
