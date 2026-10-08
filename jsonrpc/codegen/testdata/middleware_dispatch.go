package testdata

import . "github.com/CaliLuke/loom/dsl"

// MiddlewareDispatchDSL exercises the mounted dispatch boundary of every
// JSON-RPC transport shape, including the SSE GET listener.
var MiddlewareDispatchDSL = func() {
	API("dispatch", func() { JSONRPC(func() {}) })
	for _, name := range []string{"plain", "sse", "mixed", "socket"} {
		Service(name, func() {
			JSONRPC(func() {
				if name == "socket" {
					GET("/" + name)
				} else {
					POST("/" + name)
				}
			})
			if name == "plain" || name == "mixed" {
				Method("call", func() {
					Result(String)
					JSONRPC(func() {})
				})
			}
			if name == "sse" || name == "mixed" {
				Method("events/stream", func() {
					StreamingResult(String)
					JSONRPC(func() { ServerSentEvents() })
				})
			}
			if name == "socket" {
				Method("echo", func() {
					StreamingPayload(String)
					StreamingResult(String)
					JSONRPC(func() {})
				})
			}
		})
	}
}
