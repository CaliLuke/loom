package design

import . "github.com/CaliLuke/loom/dsl"

var _ = API("ticktock", func() {
	JSONRPC(func() {})
})

var streamAudit = Interceptor("streamAudit", func() {
	WriteStreamingResult(func() {
		Attribute("value")
	})
	ReadStreamingResult(func() {
		Attribute("value")
	})
})

var _ = Service("clock", func() {
	JSONRPC(func() {
		POST("/rpc")
	})

	Method("Tick", func() {
		ClientInterceptor(streamAudit)
		Payload(func() {
			ID("id", String)
			Attribute("label", String)
		})
		StreamingResult(func() {
			Attribute("value", String)
		})
		JSONRPC(func() {
			ServerSentEvents(func() { SSENotificationMethod("Tick") })
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
			ServerSentEvents(func() { SSENotificationMethod("Tock") })
		})
	})
})
