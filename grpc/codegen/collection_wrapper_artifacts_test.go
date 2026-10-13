package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestCollectionWrapperArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_COLLECTION_CONTENTS_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_COLLECTION_CONTENTS_OUTPUT for isolated artifact comparison")
	}
	for _, tc := range []struct {
		name string
		dsl  func()
	}{
		{"rich", collectionWrapperRichDSL},
		{"declarations", testdata.MessageDeclarationsDSL},
		{"recursive", recursiveNestedCollectionDSL(mapOfMapsIndex, "", nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(output, tc.name)
			require.NoError(t, os.MkdirAll(dir, 0o750))
			root := RunGRPCDSL(t, tc.dsl)
			renderGRPCModule(t, dir, "example.com/collectioncontents", root, resolveGRPCLoomSource(t))
			path := filepath.Join(dir, "go.mod")
			mod, err := os.ReadFile(path)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(path, []byte(strings.ReplaceAll(string(mod), "require github.com/CaliLuke/loom v1.0.0", "require github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261013020843-063198a2f427")), 0o600))
		})
	}
}
