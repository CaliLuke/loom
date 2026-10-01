package testdata

import . "github.com/CaliLuke/loom/dsl"

var OpenAPINamedRequestBodyExamplesDSL = func() {
	var SearchFilters = Type("SearchFilters", func() {
		Meta("openapi:component:requestBody", "SearchFiltersRequest")
		Attribute("query", String)
		Required("query")
		Example("simple", func() {
			Value(map[string]any{"query": "soup"})
		})
		Example("advanced", func() {
			Value(map[string]any{"query": "stew"})
		})
	})
	var SingletonNamedFilters = Type("SingletonNamedFilters", func() {
		Attribute("query", String)
		Required("query")
		Example("primary", func() {
			Description("Retained singleton description.")
			Meta("openapi:component:example", "  SingletonSearchExample  ")
			Meta("openapi:example:summary", "Authored singleton summary")
			Value(map[string]any{"query": "named soup"})
		})
	})
	var SingletonInlineFilters = Type("SingletonInlineFilters", func() {
		Attribute("query", String)
		Required("query")
		Example(map[string]any{"query": "inline soup"})
	})

	Service("exampleBodies", func() {
		Method("search", func() {
			Payload(func() {
				Attribute("body", SearchFilters)
				Required("body")
			})
			Result(String)
			HTTP(func() {
				POST("/search")
				Body("body")
				Response(StatusOK)
			})
		})
		Method("singletonNamed", func() {
			Payload(func() {
				Attribute("body", SingletonNamedFilters)
				Required("body")
			})
			Result(String)
			HTTP(func() {
				POST("/singleton/named")
				Body("body")
				Response(StatusOK)
			})
		})
		Method("singletonInline", func() {
			Payload(func() {
				Attribute("body", SingletonInlineFilters)
				Required("body")
			})
			Result(String)
			HTTP(func() {
				POST("/singleton/inline")
				Body("body")
				Response(StatusOK)
			})
		})
	})
}
