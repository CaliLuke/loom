package testdata

import . "github.com/CaliLuke/loom/dsl"


var OpenAPIProblemLinksAsyncDSL = func() {
	var ThreadSummary = Type("OpenAPIThreadSummary", func() {
		Attribute("thread_id", String, func() {
			Example("thr_42")
			Pattern("^thr_[0-9]+$")
		})
		Attribute("title", String, func() {
			Example("Release freeze follow-up")
		})
		Required("thread_id", "title")
	})
	var ThreadAccepted = Type("OpenAPIThreadAccepted", func() {
		Attribute("thread_id", String, func() {
			Example("thr_42")
			Pattern("^thr_[0-9]+$")
		})
		Attribute("watch_url", String, func() {
			Example("https://api.contract-surfaces.example.test/threads/thr_42/events")
		})
		Required("thread_id", "watch_url")
	})
	var ThreadEvent = Type("OpenAPIThreadEvent", func() {
		Attribute("id", String, func() {
			Example("evt_8")
		})
		Attribute("event", String, func() {
			Example("thread.message_posted")
		})
		Attribute("data", func() {
			Attribute("author", String, func() {
				Example("alice")
			})
			Attribute("preview", String, func() {
				Example("Shipping the OpenAPI cleanup next.")
			})
			Required("author", "preview")
		})
		Required("id", "event", "data")
	})

	var _ = API("contract-surfaces", func() {
		Title("Contract Surfaces API")
		Description("Exercises problem documents, workflow links, and async streaming contracts.")
		Meta("openapi:closed-objects", "true")
		Server("contract-surfaces", func() {
			Host("api", func() {
				URI("https://api.contract-surfaces.example.test")
			})
		})
	})

	Service("threadOps", func() {
		Description("Thread creation and retrieval operations.")
		Error("not_found", ProblemResult)
		Error("conflict", ProblemResult)
		Error("busy", ProblemResult)
		HTTP(func() {
			Response("not_found", StatusNotFound)
			Response("conflict", StatusConflict)
			Response("busy", StatusTooManyRequests)
		})

		Method("getThread", func() {
			NoSecurity()
			Payload(func() {
				Attribute("thread_id", String, func() {
					Example("thr_42")
				})
				Required("thread_id")
			})
			Result(ThreadSummary)
			HTTP(func() {
				GET("/threads/{thread_id}")
				Param("thread_id")
				Response(StatusOK)
			})
		})

		Method("createThread", func() {
			NoSecurity()
			Payload(func() {
				Attribute("title", String, func() {
					Example("Release freeze follow-up")
				})
				Required("title")
			})
			Result(ThreadAccepted)
			HTTP(func() {
				POST("/threads")
				Response(StatusAccepted, func() {
					Link("thread", func() {
						LinkOperation("getThread")
						LinkParam("thread_id", "$response.body#/thread_id")
					})
					Link("watch", func() {
						LinkOperation("watchThread")
						LinkParam("thread_id", "$response.body#/thread_id")
					})
				})
			})
		})

		Method("reopenThread", func() {
			NoSecurity()
			Payload(func() {
				Attribute("thread_id", String, func() {
					Example("thr_42")
				})
				Required("thread_id")
			})
			Result(ThreadAccepted)
			HTTP(func() {
				POST("/threads/{thread_id}/reopen")
				Param("thread_id")
				Response(StatusAccepted, func() {
					Link("thread", func() {
						LinkOperation("getThread")
						LinkParam("thread_id", "$response.body#/thread_id")
					})
					Link("watch", func() {
						LinkOperation("watchThread")
						LinkParam("thread_id", "$response.body#/thread_id")
					})
				})
			})
		})

		Method("archiveThread", func() {
			NoSecurity()
			Payload(func() {
				Attribute("thread_id", String, func() {
					Example("thr_42")
				})
				Required("thread_id")
			})
			Result(Empty)
			HTTP(func() {
				POST("/threads/{thread_id}/archive")
				Param("thread_id")
				Response(StatusNoContent)
			})
		})

		Method("watchThread", func() {
			NoSecurity()
			Payload(func() {
				Attribute("thread_id", String, func() {
					Example("thr_42")
				})
				Attribute("last_event_id", String, func() {
					Example("evt_7")
				})
				Required("thread_id")
			})
			StreamingResult(ThreadEvent)
			HTTP(func() {
				GET("/threads/{thread_id}/events")
				Param("thread_id")
				Param("last_event_id")
				ServerSentEvents(func() {
					SSERequestID("last_event_id")
					SSEEventID("id")
				})
			})
		})
	})

	Service("opsSocket", func() {
		Description("Bidirectional operator control channel.")

		Method("streamCommands", func() {
			NoSecurity()
			Payload(func() {
				Attribute("channel", String, func() {
					Example("deployments")
				})
				Required("channel")
			})
			StreamingPayload(func() {
				Attribute("op", String, func() {
					Example("pause")
				})
				Attribute("target", String, func() {
					Example("worker-eu-1")
				})
				Required("op", "target")
			})
			StreamingResult(func() {
				Attribute("event", String, func() {
					Example("command.accepted")
				})
				Attribute("ok", Boolean, func() {
					Example(true)
				})
				Required("event", "ok")
			})
			HTTP(func() {
				GET("/ws/ops/{channel}")
				Param("channel")
			})
		})
	})
}


