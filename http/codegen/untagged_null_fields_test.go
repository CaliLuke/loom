package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestGeneratedUntaggedNullFields(t *testing.T) {
	root := RunHTTPDSL(t, testdata.UntaggedNullFieldsDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/nullfields", root)
	harness, err := os.ReadFile("testdata/untagged_null_fields_harness.go.txt")
	require.NoError(t, err)
	source := strings.ReplaceAll(string(harness), "TRANSPORT", "http")
	source = strings.ReplaceAll(source, "SERVER_OPTIONS", "nil, nil")
	source = strings.ReplaceAll(source, "IS_RPC", "false")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nullfields_test.go"), []byte(source), 0600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}
