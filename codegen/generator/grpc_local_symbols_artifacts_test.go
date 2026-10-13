package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestGRPCLocalSymbolsExamplesCompile exercises complete example applications,
// rather than treating parsed transport-specific fragments as build evidence.
func TestGRPCLocalSymbolsExamplesCompile(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"ordinary", func() {
			testdata.GRPCLocalSymbols("ordinary")
		}},
		{"md", func() {
			testdata.GRPCLocalSymbols("md")
		}},
		{"aggregate-control", func() {
			testdata.GRPCLocalSymbolsAggregate("p", "psvcsvr")
		}},
		{"aggregate-collision", func() {
			testdata.GRPCLocalSymbolsAggregate("ordinary", "ordinarysvr")
		}},
		{"service-suffixes", func() {
			testdata.GRPCLocalSymbols("md", "mdsvc", "mdsvc2")
		}},
		{"protobuf-suffixes", func() {
			testdata.GRPCLocalSymbols("req", "reqpb", "reqpb2")
		}},
		{"hosts", testdata.GRPCLocalSymbolsHosts},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if base := os.Getenv("LOOM_691_ARTIFACTS"); base != "" {
				dir = filepath.Join(base, "example-"+tc.name)
				require.NoError(t, os.MkdirAll(dir, 0o750))
			}
			const module = "example.com/grpclocals"
			design := tc.design
			renderDesignModule(t, dir, module, design)
			root := codegen.RunDSL(t, design)
			files, err := Example(module+"/gen", []eval.Root{root})
			require.NoError(t, err)
			for _, file := range files {
				_, err := file.Render(dir)
				require.NoError(t, err, file.Path)
			}
			if os.Getenv("LOOM_691_GENERATE_ONLY") != "" {
				return
			}
			output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(dir, "go", "build", "./...")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(dir, "go", "vet", "./...")
			require.NoError(t, err, output)
		})
	}
}
