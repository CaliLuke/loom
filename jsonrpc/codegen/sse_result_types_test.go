package codegen

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
)

// TestJSONRPCSSEResultTypesStreams pins the JSON-RPC SSE stream files for
// primitive, collection, and user-type results: only the user type converts
// through a response-body constructor, and every package qualifier used by
// the emitted code is imported.
func TestJSONRPCSSEResultTypesStreams(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEResultTypesDSL)
	services := CreateJSONRPCServices(root)
	dir := t.TempDir()
	renderCodegenFiles(t, dir, SSEServerFiles("example.com/jsonrpcsseresulttypes/gen", services))

	base := filepath.Join(dir, "gen", "jsonrpc", "jsonrpcsse_result_types")
	server := readJSONRPCStreamFile(t, filepath.Join(base, "server", "stream.go"))
	require.Equal(t, 2, strings.Count(server, "body := NewStreamNoteResponseBody(result)"))
	require.Equal(t, 14, strings.Count(server, "body := result\n"))
	// Events are JSON-RPC envelopes, so primitive results are always JSON
	// encoded inside "params" or "result" and never written as raw SSE data.
	require.NotContains(t, server, "EncodeSSEData(result)")
	require.NotContains(t, server, "EncodeSSEData(body)")
	require.NotRegexp(t, `New(StreamString|StreamInt|StreamBool|StreamBytes|StreamAny|StreamArray|StreamMap)ResponseBody`, server)

	client := readJSONRPCStreamFile(t, filepath.Join(base, "client", "stream.go"))
	require.Contains(t, client, "(loom.JSONValue, error)")

	for _, path := range []string{filepath.Join(base, "server", "stream.go"), filepath.Join(base, "client", "stream.go")} {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		require.NoError(t, err)
		imports := make(map[string]bool)
		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			require.NoError(t, err)
			if spec.Name != nil {
				imports[spec.Name.Name+"="+importPath] = true
			}
		}
		require.Truef(t, imports["loom=github.com/CaliLuke/loom/pkg"], "%s must import the loom package it references", path)
	}
}

func readJSONRPCStreamFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}
