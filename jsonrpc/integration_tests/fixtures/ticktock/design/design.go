package design

import . "goa.design/goa/v3/dsl"

var _ = API("ticktock", func() {
	JSONRPC(func() {})
})

var _ = Service("clock", func() {
	JSONRPC(func() {
		POST("/rpc")
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

	Method("Tock", func() {
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
