package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
)

func TestJSONRPCSSEHandlerUsesTypedLastEventIDContextKey(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "server.go", "jsonrpc-server-handler-init")

	require.Contains(t, code, "context.WithValue(ctx, loomhttp.LastEventIDKey, lastEventID)")
	require.NotContains(t, code, `context.WithValue(ctx, "last-event-id", lastEventID)`)
}
