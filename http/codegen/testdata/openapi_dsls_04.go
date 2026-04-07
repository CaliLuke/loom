package testdata

import . "github.com/CaliLuke/loom/dsl"


var OpenAPIExplicitBodyWrapperExamplesDSL = func() {
	var SearchFilters = Type("SearchFilters", func() {
		Attribute("query", String)
		Required("query")
		Example("simple", func() {
			Value(map[string]any{"query": "soup"})
		})
		Example("advanced", func() {
			Value(map[string]any{"query": "stew"})
		})
	})
	var SearchFiltersRequestBody = Type("SearchFiltersRequestBody", func() {
		Meta("openapi:component:requestBody", "SearchFiltersRequestBody")
		Extend(SearchFilters)
	})

	Service("exampleBodyWrappers", func() {
		Method("search", func() {
			Payload(SearchFilters)
			Result(String)
			HTTP(func() {
				POST("/search")
				Body(SearchFiltersRequestBody)
				Response(StatusOK)
			})
		})
	})
}


