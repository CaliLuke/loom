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
	})
}


