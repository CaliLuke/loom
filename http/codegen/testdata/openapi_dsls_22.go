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

var MultipleMediaContractsDSL = func() {
	StreamingPartialExamplesDSL()
	envelope := Type("MediaEnvelope", func() {
		Attribute("content_type", String)
		OneOf("body", func() {
			Attribute("json", Int, func() {
				Minimum(0)
				Maximum(100)
				Example("one", 1)
				Example("two", 2)
			})
			Attribute("nullable", String, func() {
				Nullable()
				Example(Null())
			})
			Attribute("vendor", Int, func() {
				Example("vendor", 7)
			})
			Attribute("text", String, func() {
				Example("text", "hello")
			})
			Attribute("binary", Bytes, func() {
				Example("binary", []byte("bytes"))
			})
			Attribute("form", func() {
				Attribute("name", String)
				Attribute("note", String)
				Attribute("numbers", ArrayOf(Int))
				Attribute("labels", MapOf(String, String))
				Meta("openapi:encoding:labels:contentType", "application/json")
			})
			Attribute("multipart", func() {
				Attribute("name", String)
				Attribute("data", Bytes)
				Required("name", "data")
				Meta("openapi:encoding:data:contentType", "image/png")
			})
		})
		Required("body")
	})
	payload := Type("MediaRootExamples", func() {
		OneOf("body", func() {
			Attribute("number", Int)
			Attribute("text", String)
		})
		Required("body")
		Example("first", map[string]any{"body": 11})
		Example("second", map[string]any{"body": 22})
		Example("text", map[string]any{"body": "hello"})
	})
	failure := Type("MediaFailure", func() {
		OneOf("detail", func() {
			Attribute("number", Int)
			Attribute("text", String)
		})
		Required("detail")
	})
	Service("mediaContracts", func() {
		Method("fail", func() {
			Error("rejected", failure)
			HTTP(func() {
				GET("/media-contracts/fail")
				Response("rejected", StatusBadRequest, func() {
					Body("detail")
					MediaType("application/json", "number")
					MediaType("text/plain", "text")
				})
			})
		})
		Method("optional", func() {
			Payload(func() {
				OneOf("choice", func() {
					Attribute("number", Int)
					Attribute("text", String)
					Attribute("form", func() {
						Attribute("name", String)
					})
				})
			})
			Result(String)
			HTTP(func() {
				POST("/media-contracts/optional")
				Body("choice")
				MediaType("application/x-www-form-urlencoded", "form")
				MediaType("application/json", "number")
				MediaType("text/plain", "text")
				Response(StatusOK)
			})
		})

		Method("exchange", func() {
			Payload(envelope)
			Result(envelope)
			HTTP(func() {
				POST("/media-contracts")
				Body("body")
				Header("content_type:Content-Type")
				MediaType("application/json", "json")
				MediaType("application/vnd.loom+json", "vendor")
				MediaType("application/vnd.null+json", "nullable")
				MediaType("text/plain", "text")
				MediaType("*/*", "binary")
				MediaType("application/x-www-form-urlencoded", "form")
				MediaType("multipart/form-data", "multipart")
				Response(StatusOK, func() {
					Body("body")
					Header("content_type:Content-Type")
					MediaType("application/json", "json")
					MediaType("application/vnd.loom+json", "vendor")
					MediaType("application/vnd.null+json", "nullable")
					MediaType("text/plain", "text")
					MediaType("*/*", "binary")
					MediaType("application/x-www-form-urlencoded", "form")
					MediaType("multipart/form-data", "multipart")
				})
			})
		})
	})
	Service("mediaExampleSelections", func() {
		Method("rootExamples", func() {

			Payload(payload)
			Result(payload)
			HTTP(func() {
				POST("/media-root-examples")
				Body("body")
				MediaType("application/json", "number")
				MediaType("text/plain", "text")
				Response(StatusOK, func() {
					Body("body")
					MediaType("application/json", "number")
					MediaType("text/plain", "text")
				})
			})
		})
	})
}
