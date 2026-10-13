package testdata

import . "github.com/CaliLuke/loom/dsl"

// SSEViewedResultDSL exercises dynamic and fixed views, collections, and mapped
// data from the same SSE event.
var SSEViewedResultDSL = func() {
	item := ResultType("application/vnd.sse-viewed-item", func() {
		TypeName("Item")
		Attributes(func() {
			Attribute("id", String)
			Attribute("name", String)
			Attribute("secret", String)
			Attribute("count", Int)
			Attribute("required", String)
			Attribute("defaulted", String, func() {
				Default("fallback")
			})
			Required("id", "required")
		})
		View("default", func() {
			Attribute("id")
			Attribute("name")
			Attribute("count")
			Attribute("required")
			Attribute("defaulted")
		})
		View("tiny", func() {
			Attribute("id")
		})
		View("full", func() {
			Attribute("id")
			Attribute("name")
			Attribute("count")
			Attribute("required")
			Attribute("defaulted")
			Attribute("secret")
		})
	})
	Service("ViewedSSE", func() {
		Method("Watch", func() {
			StreamingResult(item)
			HTTP(func() {
				GET("/watch")
				ServerSentEvents()
			})
		})
		Method("List", func() {
			StreamingResult(CollectionOf(item))
			HTTP(func() {
				GET("/list")
				ServerSentEvents()
			})
		})
		for _, mapping := range []struct {
			method, path, field string
		}{
			{"Mapped", "/mapped", "name"},
			{"Number", "/number", "count"},
			{"Required", "/required", "required"},
			{"Defaulted", "/defaulted", "defaulted"},
		} {
			Method(mapping.method, func() {
				StreamingResult(item)
				HTTP(func() {
					GET(mapping.path)
					ServerSentEvents(mapping.field)
				})
			})
		}
		Method("Full", func() {
			StreamingResult(item, func() {
				View("full")
			})
			HTTP(func() {
				GET("/full")
				ServerSentEvents()
			})
		})
	})
}
