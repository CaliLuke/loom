package testdata

import (
	. "goa.design/goa/v3/dsl"
)

var JSONRPCSSEStringDSL = func() {
	API("jsonrpc-sse-test", func() {
		JSONRPC(func() {})
	})
	Service("JSONRPCSSEStringService", func() {
		JSONRPC(func() {
			POST("/stream")
		})
		Method("Stream", func() {
			Payload(func() {
				ID("id", String, "Request ID")
			})
			StreamingResult(String)
			JSONRPC(func() {
				ServerSentEvents()
			})
		})
	})
}

var JSONRPCSSEObjectDSL = func() {
	API("jsonrpc-sse-test", func() {
		JSONRPC(func() {})
	})
	Service("JSONRPCSSEObjectService", func() {
		JSONRPC(func() {
			POST("/stream")
		})
		Method("Stream", func() {
			Payload(func() {
				ID("id", String, "Request ID")
				Attribute("last_event_id", String, "Last event ID")
			})
			StreamingResult(func() {
				ID("id", String, "Event ID")
				Attribute("data", String, "Event data")
			})
			JSONRPC(func() {
				ServerSentEvents(func() {
					SSERequestID("last_event_id")
					SSEEventID("id")
				})
			})
		})
	})
}

var JSONRPCSSEEventsStreamDSL = func() {
	API("jsonrpc-sse-events-stream-test", func() {
		JSONRPC(func() {})
	})
	Service("JSONRPCSSEEventsStreamService", func() {
		JSONRPC(func() {
			POST("/rpc")
		})
		Method("events/stream", func() {
			Payload(func() {
				ID("id", String, "Request ID")
			})
			StreamingResult(func() {
				Attribute("value", String)
			})
			JSONRPC(func() {
				ServerSentEvents()
			})
		})
	})
}
