package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
)

func TestJSONRPCSSEUsesNamespacedDefaultNotificationMethod(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEObjectDSL)
	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "sse.go", "jsonrpc-server-sse-stream-impl")

	require.Contains(t, code, `"method":  "JSONRPCSSEObjectService/stream.event"`)
	require.NotContains(t, code, `"method":  "Stream"`)
}

func TestJSONRPCSSEUsesDesignedNotificationMethod(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcSSENotificationMethodDSL)
	code := fileSectionCode(t, ServerFiles("", CreateJSONRPCServices(root)), "sse.go", "jsonrpc-server-sse-stream-impl")

	require.Contains(t, code, `"method":  "notifications/progress"`)
	require.NotContains(t, code, `/stream.event`)
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
