package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestGeneratedRequiredOccurrences(t *testing.T) {
	root := RunHTTPDSL(t, testdata.RequiredOccurrenceDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/occurrences", root)
	harness, err := os.ReadFile("testdata/required_occurrence_harness.go.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "occurrences_test.go"), harness, 0600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}
