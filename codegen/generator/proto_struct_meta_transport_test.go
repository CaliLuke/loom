package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	codegentestdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestProtoStructMetaGeneratedCodeCompiles builds and vets the service, gRPC
// transport and gRPC CLI packages generated for types with struct:pkg:path
// and struct:name:proto metadata. The protocol buffer messages must be
// referenced from the pb package, never from the struct:pkg:path package.
func TestProtoStructMetaGeneratedCodeCompiles(t *testing.T) {
	dir := buildGeneratedModule(t, "example.com/protostructmeta", codegentestdata.ProtoStructMetaDSL)
	_, err := os.Stat(filepath.Join(dir, "gen", "grpc", "cli", "test_api", "cli.go"))
	require.NoError(t, err)
	output, err := testingx.RunCmd(dir, "go", "vet", "./...")
	require.NoError(t, err, output)
}
