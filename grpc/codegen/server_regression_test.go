package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestServerFilesUseGoaGRPCHandlerAliases(t *testing.T) {
	t.Run("unary handlers use goagrpc alias", func(t *testing.T) {
		code := renderGRPCServerFile(t, testdata.UnaryRPCsDSL)
		require.Contains(t, code, "goagrpc.UnaryHandler")
		require.NotContains(t, code, " grpc.UnaryHandler")
	})

	t.Run("stream handlers use goagrpc alias", func(t *testing.T) {
		code := renderGRPCServerFile(t, testdata.ServerStreamingRPCDSL)
		require.Contains(t, code, "goagrpc.StreamHandler")
		require.NotContains(t, code, " grpc.StreamHandler")
	})
}

func renderGRPCServerFile(t *testing.T, dsl func()) string {
	t.Helper()

	root := RunGRPCDSL(t, dsl)
	services := CreateGRPCServices(root)
	files := ServerFiles("", services)
	require.NotEmpty(t, files)

	for _, file := range files {
		if filepath.Base(file.Path) != "server.go" {
			continue
		}
		renderedPath, err := file.Render(t.TempDir())
		require.NoError(t, err)

		rendered, err := os.ReadFile(renderedPath)
		require.NoError(t, err)
		return string(rendered)
	}

	t.Fatalf("server.go not found in generated files")
	return ""
}
