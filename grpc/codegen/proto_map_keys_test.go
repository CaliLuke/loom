package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr/testdata"
)

func TestProtoMapKeys(t *testing.T) {
	code := protoFileCode(t, testdata.GRPCMapKeys)
	for _, key := range []string{"bool", "string", "sint32", "sint64", "uint32", "uint64"} {
		require.Contains(t, code, "map<"+key+", google.protobuf.Value>")
	}
	require.Contains(t, code, "map<string, Node> children")
	path := codegen.CreateTempFile(t, code)
	require.NoError(t, protoc(defaultProtocCmd, path, nil))
	dir := t.TempDir()
	renderGRPCModule(t, dir, "example.com/mapkeys", RunGRPCDSL(t, testdata.GRPCMapKeys), resolveGRPCLoomSource(t))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "vet", "./...")
}
