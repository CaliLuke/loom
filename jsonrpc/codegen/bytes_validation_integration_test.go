package codegen

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	codegentestdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/bytes_validation_jsonrpc_test.go.txt
var bytesValidationJSONRPCHarness string

// TestBytesValidationGeneratedJSONRPC compiles request, client response and
// service view validation with retained named byte slices.
func TestBytesValidationGeneratedJSONRPC(t *testing.T) {
	root := RunJSONRPCDSL(t, codegentestdata.BytesValidationJSONRPCDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/bytesvalidation", root)
	data := servicecodegen.NewServicesData(root)
	for _, svc := range root.Services {
		if views := servicecodegen.ViewsFile("example.com/bytesvalidation/gen", svc, data); views != nil {
			_, err := views.Render(dir)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bytes_test.go"), []byte(bytesValidationJSONRPCHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "build", "./...")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}
