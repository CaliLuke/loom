package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
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

// JSONRPCSSEResultTypesDSL declares one JSON-RPC SSE method per primitive,
// collection, and user-type streaming result so generated server and client
// code can be compiled and exercised for each event shape.
var JSONRPCSSEResultTypesDSL = func() {
	API("jsonrpc-sse-result-types", func() {
		JSONRPC(func() {})
	})
	var Note = Type("Note", func() {
		Attribute("text", String)
		Required("text")
	})
	Service("JSONRPCSSEResultTypes", func() {
		JSONRPC(func() {
			POST("/rpc")
		})
		streamMethod := func(name string, result any) {
			Method(name, func() {
				Payload(func() {
					ID("id", String, "Request ID")
				})
				StreamingResult(result)
				JSONRPC(func() {
					ServerSentEvents()
				})
			})
		}
		streamMethod("StreamString", String)
		streamMethod("StreamInt", Int)
		streamMethod("StreamBool", Boolean)
		streamMethod("StreamBytes", Bytes)
		streamMethod("StreamAny", Any)
		streamMethod("StreamArray", ArrayOf(String))
		streamMethod("StreamMap", MapOf(String, Int))
		streamMethod("StreamNote", Note)
	})
}
