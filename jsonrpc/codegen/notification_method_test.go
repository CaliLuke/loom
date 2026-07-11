package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
)

func TestJSONRPCSSEUsesNamespacedDefaultNotificationMethod(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	services := CreateJSONRPCServices(root)
	serverCode := fileSectionCode(t, ServerFiles("", services), "sse.go", "jsonrpc-server-sse-stream-impl")
	clientCode := fileSectionCode(t, SSEServerFiles("", services), "stream.go", "jsonrpc-sse-client-stream")

	require.Contains(t, serverCode, `"method":  "JSONRPCSSEObjectService/stream.event"`)
	require.Contains(t, clientCode, `notification.Method != "JSONRPCSSEObjectService/stream.event"`)
	require.NotContains(t, serverCode, `"method":  "Stream"`)
}

func TestJSONRPCSSEUsesDesignedNotificationMethod(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcSSENotificationMethodDSL)
	services := CreateJSONRPCServices(root)
	serverCode := fileSectionCode(t, ServerFiles("", services), "sse.go", "jsonrpc-server-sse-stream-impl")
	clientCode := fileSectionCode(t, SSEServerFiles("", services), "stream.go", "jsonrpc-sse-client-stream")

	require.Contains(t, serverCode, `"method":  "notifications/progress"`)
	require.Contains(t, clientCode, `notification.Method != "notifications/progress"`)
	require.NotContains(t, serverCode, `/stream.event`)
}

var jsonrpcSSENotificationMethodDSL = func() {
	dsl.Service("JSONRPCProgress", func() {
		dsl.JSONRPC(func() { dsl.POST("/rpc") })
		dsl.Method("tools/call", func() {
			dsl.Payload(func() { dsl.ID("id", dsl.String) })
			dsl.StreamingResult(dsl.String)
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents(func() {
					dsl.SSENotificationMethod("notifications/progress")
				})
			})
		})
	})
}
