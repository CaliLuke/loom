package codegen

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	. "github.com/CaliLuke/loom/dsl"
)

func TestJSONRPCWebSocketUsesSharedRuntimeStream(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcWebSocketRuntimeDSL)
	services := CreateJSONRPCServices(root)

	serverCode := renderedJSONRPCWebSocketFile(t, ServerFiles("", services), "server")
	require.Contains(t, serverCode, "conn *loomhttp.WebSocketStream")
	require.Contains(t, serverCode, "requestHasID bool")
	require.Contains(t, serverCode, "requestHasID: req.HasID")
	require.Contains(t, serverCode, "jsonrpc.CompleteStream(ctx, w.requestHasID, w.requestID, body")
	require.Contains(t, serverCode, "jsonrpc.CompleteStreamError(ctx, w.requestHasID")
	require.NotContains(t, serverCode, "WriteControl(")
	require.NotContains(t, serverCode, `"github.com/gorilla/websocket"`)
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-websocket-server-file.golden"), serverCode)
	serverHandlerCode := renderedJSONRPCFile(t, ServerFiles("", services), "server.go", "server")
	require.Contains(t, serverHandlerCode, "jsonrpc.ServeWebSocket(")
	require.Contains(t, serverHandlerCode, "conn:           conn")

	clientCode := renderedJSONRPCWebSocketFile(t, ClientFiles("", services), "client")
	require.Contains(t, clientCode, "ws          *loomhttp.WebSocketStream")
	require.NotContains(t, clientCode, "writeMu")
	testutil.AssertGo(t, filepath.Join("testdata", "golden", "jsonrpc-websocket-client-file.golden"), clientCode)
	clientEndpointCode := renderedJSONRPCFile(t, ClientFiles("", services), "client.go", "client")
	require.Contains(t, clientEndpointCode, "ws:      loomhttp.NewWebSocketStream(ws)")
}

func renderedJSONRPCWebSocketFile(t *testing.T, files []*codegen.File, side string) string {
	return renderedJSONRPCFile(t, files, "websocket.go", side)
}

func renderedJSONRPCFile(t *testing.T, files []*codegen.File, base string, side string) string {
	t.Helper()
	for _, file := range files {
		if filepath.Base(file.Path) != base || filepath.Base(filepath.Dir(file.Path)) != side {
			continue
		}
		var b strings.Builder
		for _, section := range file.AllSections() {
			require.NoError(t, section.Write(&b))
		}
		return b.String()
	}
	t.Fatalf("missing %s %s", side, base)
	return ""
}

func jsonrpcWebSocketRuntimeDSL() {
	var Request = Type("JSONRPCWebSocketRuntimeRequest", func() {
		Attribute("value", String)
	})
	var Result = Type("JSONRPCWebSocketRuntimeResult", func() {
		Attribute("value", String)
	})

	API("jsonrpc-websocket-runtime-test", func() {
		JSONRPC(func() {})
	})
	Service("JSONRPCWebSocketRuntimeService", func() {
		JSONRPC(func() {
			GET("/rpc")
		})
		Method("Stream", func() {
			StreamingPayload(Request)
			StreamingResult(Result)
			JSONRPC(func() {})
		})
	})
}
