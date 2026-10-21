package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestGeneratedCLIArgumentOrder(t *testing.T) {
	for _, tc := range []struct {
		transport string
		shape     string
	}{
		{"http", "combined"},
		{"http", "split"},
		{"http", "single"},
		{"http", "multipart"},
		{"http", "interceptors"},
		{"http", "stream"},
		{"http", "subset"},
		{"http", "file-only"},
		{"jsonrpc", "interceptors"},
		{"jsonrpc", "rpc-stream"},
	} {
		t.Run(tc.transport+"/"+tc.shape, func(t *testing.T) {
			dir := t.TempDir()
			if output := os.Getenv("LOOM_CLI_ORDER_OUTPUT"); output != "" {
				dir = filepath.Join(output, tc.transport, tc.shape)
				require.NoError(t, os.MkdirAll(dir, 0o700))
			}
			codegen.RunDSL(t, testdata.CLIArgumentOrderDSL(tc.transport, tc.shape))
			mod := fmt.Sprintf("module example.com/order\n\ngo 1.27.3\n\nrequire github.com/CaliLuke/loom v1.0.0\nreplace github.com/CaliLuke/loom => %s\n", loomModuleSource(t))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o600))
			_, err := Generate(dir, "gen", "github.com/CaliLuke/loom/codegen/generator", false)
			require.NoError(t, err)
			_, err = Generate(dir, "example", "github.com/CaliLuke/loom/codegen/generator", false)
			require.NoError(t, err)
			if os.Getenv("LOOM_CLI_ORDER_GENERATE_ONLY") != "" {
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
