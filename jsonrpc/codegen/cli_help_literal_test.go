package codegen

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	httpdata "github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestJSONRPCCLIHelpLiteralsGenerated(t *testing.T) {
	for _, special := range []bool{false, true} {
		t.Run(strconv.FormatBool(special), func(t *testing.T) {
			root := RunJSONRPCDSL(t, httpdata.CLIHelpLiteralDSL("jsonrpc", special))
			dir := t.TempDir()
			if output := os.Getenv("LOOM_CLI_HELP_OUTPUT"); output != "" {
				dir = filepath.Join(output, "jsonrpc", strconv.FormatBool(special))
				require.NoError(t, os.MkdirAll(dir, 0o700))
			}
			renderJSONRPCModule(t, dir, "example.com/help", root)
			renderCodegenFiles(t, dir, ClientCLIFiles("example.com/help/gen", CreateJSONRPCServices(root)))
			text := "ordinary text"
			if special {
				text = httpdata.CLIHelpLiteralText
			}
			harness := strings.ReplaceAll(httpdata.CLIHelpLiteralHarness, "EXPECTED_TEXT", strconv.Quote(text))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "gen/jsonrpc/cli/literalcli/cli_help_test.go"), []byte(harness), 0o600))
			runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
			runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
			runGoJSONRPCTestCommand(t, dir, "vet", "./...")
		})
	}
}
