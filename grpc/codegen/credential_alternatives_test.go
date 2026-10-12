package codegen

import (
	"os"
	"path/filepath"
	"testing"

	fixture "github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/stretchr/testify/require"
)

func TestCredentialAlternativesGenerated(t *testing.T) {
	root := RunGRPCDSL(t, fixture.CredentialAlternativesDSL("grpc"))
	dir := t.TempDir()
	renderGRPCResponseContractModule(t, dir, "example.com/alternatives", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "alternatives_test.go"), []byte(fixture.CredentialAlternativesGRPCHarness()), 0600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "test", "-race", "-run", "^TestCredentialAlternatives$", "./...")
}
