package codegen

import (
	"os"
	"path/filepath"
	"testing"

	fixture "github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/stretchr/testify/require"
)

func TestCredentialAlternativesGenerated(t *testing.T) {
	root := RunHTTPDSL(t, fixture.CredentialAlternativesDSL("http"))
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/alternatives", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "alternatives_test.go"), []byte(fixture.CredentialAlternativesHTTPHarness("http")), 0600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-race", "./...")
}
