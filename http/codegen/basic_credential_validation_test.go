package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	servicedata "github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestBasicCredentialValidationGenerated(t *testing.T) {
	root := RunHTTPDSL(t, servicedata.BasicCredentialConstraintsDSL("http"))
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/basic", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "basic_test.go"), []byte(servicedata.BasicCredentialConstraintsHarness("http")), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-race", ".")
}
