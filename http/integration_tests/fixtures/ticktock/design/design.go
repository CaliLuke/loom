package design

import . "github.com/CaliLuke/loom/dsl"

var _ = API("ticktock", func() {
	Title("Tick Tock SSE Fixture")
})

var TickTockEvent = Type("TickTockEvent", func() {
	Attribute("event", String)
	Attribute("data", String)
	Required("event", "data")
})

var _ = Service("clock", func() {
	Method("Tick", func() {
		StreamingResult(TickTockEvent)
		HTTP(func() {
			GET("/tick")
			ServerSentEvents(func() {
				SSEEventType("event")
				SSEEventData("data")
			})
		})
	})

	Method("Tock", func() {
		StreamingResult(TickTockEvent)
		HTTP(func() {
			GET("/tock")
			ServerSentEvents(func() {
				SSEEventType("event")
				SSEEventData("data")
			})
		})
	})

	Method("Guarded", func() {
		Payload(func() {
			Attribute("token", String)
		})
		Error("unauthorized")
		StreamingResult(TickTockEvent)
		HTTP(func() {
			GET("/guarded")
			Param("token")
			Response("unauthorized", StatusUnauthorized)
			ServerSentEvents(func() {
				SSEEventType("event")
				SSEEventData("data")
			})
		})
	})
})
