package testdata

import . "github.com/CaliLuke/loom/dsl"

var StreamingPartialExamplesDSL = func() {
	var RealtimeChunk = Type("RealtimeChunk", Bytes, func() {
		Example([]byte{0, 255})
	})
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
		Attribute("attachment", RealtimeChunk)
		Attribute("note", String, func() {
			Nullable()
			Example(Null())
		})
		Required("ts", "event")
		Example(Val{"ts": 1})
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
		Method("metadata", func() {
			NoSecurity()
			Payload(func() {
				Attribute("chunks", ArrayOf(RealtimeChunk), func() {
					Example([][]byte{{0, 255}})
				})
			})
			Result(func() {
				Attribute("checksum", RealtimeChunk)
				Attribute("label", String)
				Required("checksum", "label")
			})
			HTTP(func() {
				GET("/metadata")
				Param("chunks")
				Response(StatusOK, func() {
					Header("checksum:X-Checksum")
				})
			})
		})
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

var MultipleSchemaExamplesDSL = func() {
	StreamingPartialExamplesDSL()
	colors := Type("ExampleColors", String, func() {
		Nullable()
		for _, example := range []struct {
			name  string
			value any
		}{{"empty", Null()}, {"red", "red"}, {"blue", "blue"}, {"blue-again", "blue"}} {
			Example(example.name, func() {
				Meta("openapi:example:scope", "schema")
				Value(example.value)
			})
		}
		Example("media", func() {
			Meta("openapi:example:scope", "media")
			Value("green")
		})
	})
	Service("schemaExamples", func() {
		Method("echo", func() {
			Payload(func() {
				Attribute("color", colors)
				Attribute("bytes", Bytes, func() {
					Example("first", []byte("a"))
					Example("second", []byte("b"))
				})
			})
			Result(colors)
			HTTP(func() {
				POST("/schema-examples")
				Response(StatusOK)
			})
		})
	})
}
