package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestClientStreamCleanupReadiness(t *testing.T) {
	for _, transport := range []string{"http-sse", "jsonrpc-sse", "jsonrpc-ws"} {
		t.Run(transport, func(t *testing.T) {
			root := runDSL(t, func() {
				dsl.API("streamcontract", func() {
					if transport != "http-sse" {
						dsl.JSONRPC(func() {
						})
					}
				})
				dsl.Service("events", func() {
					if transport != "http-sse" {
						dsl.JSONRPC(func() {
							if transport == "jsonrpc-ws" {
								dsl.GET("/rpc")
							} else {
								dsl.POST("/rpc")
							}
						})
					}
					dsl.Method("Watch", func() {
						dsl.StreamingResult(func() {
							dsl.Attribute("message", dsl.String)
						})
						switch transport {
						case "http-sse":
							dsl.HTTP(func() {
								dsl.GET("/watch")
								dsl.ServerSentEvents()
							})
						case "jsonrpc-sse":
							dsl.JSONRPC(func() {
								dsl.ServerSentEvents()
							})
						case "jsonrpc-ws":
							dsl.JSONRPC(func() {
							})
						}
					})
				})
			})
			method := NewServicesData(root).Get("events").Methods[0]
			require.True(t, method.ClientStream.MustClose, "client interface and interceptor wrapper must retain transport Close")
		})
	}
}
