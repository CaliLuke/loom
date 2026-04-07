package testdata

import . "github.com/CaliLuke/loom/dsl"


var CollabStreamsDSL = func() {
	var ThreadSummary = Type("ThreadSummary", func() {
		Attribute("thread_id", String, func() {
			Example("thr_42")
			Pattern("^thr_[0-9]+$")
		})
		Attribute("title", String, func() {
			Example("Release freeze follow-up")
		})
		Attribute("participants", ArrayOf(String), func() {
			Example([]string{"alice", "bob"})
		})
		Required("thread_id", "title")
	})
	var _ = API("collab-streams", func() {
		Title("Collaboration Streams API")
		Description("Collaborative thread operations exposed through HTTP and SSE.")
		Meta("openapi:closed-objects", "true")
		License(func() {
			Name("Apache-2.0")
			URL("https://www.apache.org/licenses/LICENSE-2.0.html")
		})
		Server("collab-streams", func() {
			Host("api", func() {
				URI("https://api.collab-streams.example.test")
			})
		})
	})
	Service("collabStreams", func() {
		Description("Thread collaboration entry points.")

		Method("getThread", func() {
			NoSecurity()
			Payload(func() {
				Attribute("thread_id", String, func() {
					Example("thr_42")
					Pattern("^thr_[0-9]+$")
				})
				Required("thread_id")
			})
			Result(ThreadSummary)
			Error("not_found")
			HTTP(func() {
				GET("/threads/{thread_id}")
				Param("thread_id")
				Response(StatusOK)
				Response("not_found", StatusNotFound)
			})
		})

		Method("watchThread", func() {
			NoSecurity()
			Payload(func() {
				Attribute("thread_id", String, func() {
					Example("thr_42")
					Pattern("^thr_[0-9]+$")
				})
				Attribute("last_event_id", String, func() {
					Example("evt_7")
				})
				Required("thread_id")
			})
			StreamingResult(func() {
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
			Error("busy")
			HTTP(func() {
				GET("/threads/{thread_id}/events")
				Param("thread_id")
				Param("last_event_id")
				ServerSentEvents(func() {
					SSERequestID("last_event_id")
					SSEEventID("id")
				})
				Response("busy", StatusTooManyRequests)
			})
		})
	})
}


