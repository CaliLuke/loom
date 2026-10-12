package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	servicedata "github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestBasicCLIDefaultsGenerated(t *testing.T) {
	root := RunJSONRPCDSL(t, servicedata.BasicCLIDefaultsDSL("jsonrpc"))
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/basic", root)
	renderCodegenFiles(t, dir, ClientCLIFiles("example.com/basic/gen", CreateJSONRPCServices(root)))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cli_defaults_test.go"), []byte(strings.NewReplacer("TRANSPORT", "jsonrpc", "MULTIPARTARGS", "").Replace(servicedata.BasicCLIDefaultsHarness)), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "build", "./...")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
}
