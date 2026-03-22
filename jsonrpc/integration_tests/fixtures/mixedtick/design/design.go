package design

import . "github.com/CaliLuke/loom/dsl"

var _ = API("mixedtick", func() {
	JSONRPC(func() {})
})

var _ = Service("clock", func() {
	JSONRPC(func() {
		POST("/rpc")
	})

	Method("Initialize", func() {
		Payload(func() {
			ID("id", String)
		})
		Result(func() {
			ID("id", String)
			Attribute("protocol_version", String)
		})
		JSONRPC(func() {})
	})

	Method("Tick", func() {
		Payload(func() {
			ID("id", String)
		})
		StreamingResult(func() {
			Attribute("value", String)
		})
		JSONRPC(func() {
			ServerSentEvents()
		})
	})
})
