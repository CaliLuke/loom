package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestJSONRPCResponseMetadataPreservedInHandler(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		dsl.API("jsonrpc-response-metadata-test", func() {
			dsl.JSONRPC(func() {})
		})
		dsl.Service("session", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			dsl.Method("initialize", func() {
				dsl.Payload(func() {
					dsl.ID("id", dsl.String, "Request ID")
				})
				dsl.Result(func() {
					dsl.ID("id", dsl.String, "Response ID")
					dsl.Attribute("protocol_version", dsl.String)
					dsl.Attribute("session_id", dsl.String)
					dsl.Attribute("refresh", dsl.String)
					dsl.Required("protocol_version", "session_id", "refresh")
				})
				dsl.JSONRPC(func() {
					dsl.Response(func() {
						dsl.Header("session_id:Mcp-Session-Id")
						dsl.Cookie("refresh:ak-refresh")
					})
				})
			})
		})
	})

	services := CreateJSONRPCServices(root)
	serverFiles := ServerFiles("", services)
	require.NotEmpty(t, serverFiles)

	var serverCode string
	for _, f := range serverFiles {
		if filepath.Base(f.Path) != "server.go" || filepath.Base(filepath.Dir(f.Path)) != "server" {
			continue
		}
		for _, s := range f.AllSections() {
			serverCode += renderSectionSource(s)
		}
	}

	require.Contains(t, serverCode, "type jsonrpcResponseCapture struct")
	require.Contains(t, serverCode, "encodeResponse := EncodeInitializeResponse(encoder)")
	require.Contains(t, serverCode, "copyJSONRPCResponseMetadata(w, capture)")
	require.Contains(t, serverCode, `case "Content-Length", "Content-Type", "Transfer-Encoding":`)
	require.Contains(t, serverCode, `result = jsontext.Value(capture.body.Bytes())`)
}

func TestJSONRPCNotificationHandlerDoesNotBindUnusedResult(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		dsl.API("jsonrpc-notification-test", func() {
			dsl.JSONRPC(func() {})
		})
		dsl.Service("events", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			dsl.Method("notify_status_update", func() {
				dsl.Payload(func() {
					dsl.Attribute("message", dsl.String)
					dsl.Required("message")
				})
				dsl.JSONRPC(func() {})
			})
		})
	})

	services := CreateJSONRPCServices(root)
	serverFiles := ServerFiles("", services)
	require.NotEmpty(t, serverFiles)

	var serverCode string
	for _, f := range serverFiles {
		if filepath.Base(f.Path) != "server.go" || filepath.Base(filepath.Dir(f.Path)) != "server" {
			continue
		}
		for _, s := range f.AllSections() {
			serverCode += renderSectionSource(s)
		}
	}

	require.Contains(t, serverCode, "_, err = endpoint(ctx, params)")
	require.NotContains(t, serverCode, "res, err := endpoint(ctx, params)")
}
