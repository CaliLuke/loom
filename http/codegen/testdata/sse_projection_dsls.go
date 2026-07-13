package testdata

import . "github.com/CaliLuke/loom/dsl"

// SSEProjectionDSL defines one canonical event with legacy-flat and modern
// union-bearing SSE JSON projections selected by the SSE event discriminator.
var SSEVariantProjectionDSL = func() {
	var Payload = Type("ProjectionPayload", func() {
		Attribute("id", String)
		Required("id")
	})
	var Event = ResultType("application/vnd.sse-projection-event", func() {
		TypeName("ProjectionEvent")
		Attributes(func() {
			Attribute("event_type", String)
			Attribute("sequence", Int)
			Attribute("type", String)
			Attribute("payload", Payload)
			OneOf("event", func() {
				Attribute("updated", Payload)
				Attribute("deleted", Payload)
			})
			Required("event_type", "sequence")
		})
		View("legacy", func() {
			Attribute("event_type")
			Attribute("sequence")
			Attribute("type")
			Attribute("payload")
			ViewRequired("type", "payload")
		})
		View("updated", func() {
			Attribute("event_type")
			Attribute("sequence")
			Attribute("event")
			ViewRequired("event")
		})
	})
	Service("SSEProjection", func() {
		Method("Watch", func() {
			StreamingResult(Event)
			HTTP(func() {
				GET("/events")
				ServerSentEvents(func() {
					SSEEventType("event_type")
					SSEProjection("legacy", "legacy")
					SSEProjection("updated", "updated")
				})
			})
		})
	})
}
