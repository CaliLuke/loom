package codegen

import (
	"os"
	"path/filepath"
	"testing"

	fixture "github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/stretchr/testify/require"
)

func TestCredentialAlternativesGenerated(t *testing.T) {
	root := RunJSONRPCDSL(t, fixture.CredentialAlternativesDSL("jsonrpc"))
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/alternatives", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "alternatives_test.go"), []byte(fixture.CredentialAlternativesHTTPHarness("jsonrpc")), 0600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
}
