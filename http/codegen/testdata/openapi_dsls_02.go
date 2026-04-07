package testdata

import . "github.com/CaliLuke/loom/dsl"


var OpenAPIExplicitReusableComponentNamesDSL = func() {
	var SearchFilters = Type("SearchFilters", func() {
		Meta("openapi:component:requestBody", "SearchFiltersRequest")
		Attribute("query", String, func() {
			Example("soup")
		})
		Required("query")
	})

	Service("componentNames", func() {
		Method("searchRecipes", func() {
			Payload(func() {
				Attribute("widgetID", String, func() {
					Meta("openapi:component:parameter", "WidgetIDParam")
					Example("widget-123")
				})
				Attribute("body", SearchFilters)
				Required("widgetID", "body")
			})
			Result(String)
			HTTP(func() {
				POST("/recipes/{widgetID}/search")
				Param("widgetID")
				Body("body")
				Response(StatusOK)
			})
		})

		Method("searchGadgets", func() {
			Payload(func() {
				Attribute("widgetID", String, func() {
					Meta("openapi:component:parameter", "WidgetIDParam")
					Example("widget-123")
				})
				Attribute("body", SearchFilters)
				Required("widgetID", "body")
			})
			Result(String)
			HTTP(func() {
				POST("/gadgets/{widgetID}/search")
				Param("widgetID")
				Body("body")
				Response(StatusOK)
			})
		})
	})
}


