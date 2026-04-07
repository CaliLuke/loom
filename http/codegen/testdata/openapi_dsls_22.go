package testdata

import . "github.com/CaliLuke/loom/dsl"


var StreamingPartialExamplesDSL = func() {
	var RealtimeSSEEvent = Type("RealtimeSSEEvent", func() {
		Attribute("event", String, func() {
			Example("abc123")
		})
		Attribute("data", func() {
			Attribute("message", String)
			Required("message")
		})
		Required("event", "data")
	})
	var RealtimeEnvelope = Type("RealtimeEnvelope", func() {
		Attribute("ts", Int64, func() {
			Example(1)
		})
		Attribute("event", String)
		Required("ts", "event")
	})
	var _ = API("streaming-partial-examples", func() {
		Title("Streaming Partial Examples API")
		Description("Exercises suppression of invalid synthetic examples for streaming responses.")
		License(func() {
			Name("Apache-2.0")
			URL("https://www.apache.org/licenses/LICENSE-2.0.html")
		})
		Server("streaming-partial-examples", func() {
			Host("api", func() {
				URI("https://api.streaming-partial-examples.example.test")
			})
		})
	})
	Service("streamingPartialExamples", func() {
		Method("events", func() {
			NoSecurity()
			StreamingResult(RealtimeSSEEvent)
			HTTP(func() {
				GET("/events")
				ServerSentEvents()
			})
		})
		Method("projectSocket", func() {
			NoSecurity()
			Payload(func() {
				Attribute("projectID", String, func() {
					Example("proj_1")
				})
				Required("projectID")
			})
			StreamingResult(RealtimeEnvelope)
			HTTP(func() {
				GET("/ws/projects/{projectID}")
				Param("projectID")
			})
		})
	})
}


