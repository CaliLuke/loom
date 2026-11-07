package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestGeneratedUntaggedNullFields(t *testing.T) {
	root := RunGRPCDSL(t, testdata.UntaggedNullFieldsDSL)
	dir := t.TempDir()
	renderGRPCModule(t, dir, "example.com/grpcnullfields", root, resolveGRPCLoomSource(t))
	harness, err := os.ReadFile("testdata/untagged_null_fields_harness.go.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nullfields_test.go"), harness, 0600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "test", ".", "./gen/...")
}
