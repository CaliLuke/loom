package codegen

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

//go:embed testdata/media_contract_harness.go.txt
var mediaContractHarness string

func TestGeneratedMediaContracts(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MultipleMediaContractsDSL)
	directory := t.TempDir()
	renderHTTPModule(t, directory, "example.com/mediacontracts", root)
	require.NoError(t, os.WriteFile(filepath.Join(directory, "media_contract_test.go"), []byte(mediaContractHarness), 0o600))
	runGoCommand(t, directory, "mod", "tidy")
	runGoCommand(t, directory, "test", "-race", "-count=1", ".")
}
