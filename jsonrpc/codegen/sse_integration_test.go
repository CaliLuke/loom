package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
)

func TestJSONRPCSSEIntegration(t *testing.T) {
	// Run the DSL
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	services := CreateJSONRPCServices(root)

	// Generate all files
	serverFiles := ServerFiles("", services)
	clientFiles := ClientFiles("", services)
	sseFiles := SSEServerFiles("", services)

	// Combine all files
	allFiles := make([]*codegen.File, 0, len(serverFiles)+len(clientFiles)+len(sseFiles))
	allFiles = append(allFiles, serverFiles...)
	allFiles = append(allFiles, clientFiles...)
	allFiles = append(allFiles, sseFiles...)

	// Create temp directory
	tmpDir := t.TempDir()

	// Write all files
	for _, f := range allFiles {
		_, err := f.Render(tmpDir)
		require.NoError(t, err)
	}

	// Try to compile the generated code
	// This would require setting up go.mod, etc. so we'll just verify
	// that files were generated with expected content

	// Check key files exist
	serverStreamPath := filepath.Join(tmpDir, "gen/jsonrpc/jsonrpcsse_object_service/server/stream.go")
	require.FileExists(t, serverStreamPath)

	clientStreamPath := filepath.Join(tmpDir, "gen/jsonrpc/jsonrpcsse_object_service/client/stream.go")
	require.FileExists(t, clientStreamPath)

	// Read and verify server stream has JSON-RPC notification code
	serverContent, err := os.ReadFile(serverStreamPath)
	require.NoError(t, err)
	require.Contains(t, string(serverContent), "JSON-RPC notification")
	require.Contains(t, string(serverContent), `"jsonrpc": "2.0"`)
	require.Contains(t, string(serverContent), "*loomhttp.SSEStreamWriter")
	require.Contains(t, string(serverContent), "func (s *StreamServerStream) SendComment")

	// Read and verify client stream has JSON-RPC decoding
	clientContent, err := os.ReadFile(clientStreamPath)
	require.NoError(t, err)
	require.Contains(t, string(clientContent), "decodeResult")
	require.Contains(t, string(clientContent), `case "notification":`)
	require.Contains(t, string(clientContent), `if notification.JSONRPC != "2.0"`)
}
