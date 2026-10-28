package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

var OpenAPIExplicitReusableComponentNamesDSL = openAPIExplicitReusableComponentNamesDSL(false)

// OpenAPIReferencedComponentsDSL exercises imported reusable transport identities.
var OpenAPIReferencedComponentsDSL = openAPIExplicitReusableComponentNamesDSL(true)

func openAPIExplicitReusableComponentNamesDSL(references bool) func() {
	return func() {
		var SearchFilters = Type("SearchFilters", func() {
			Meta("openapi:component:requestBody", "SearchFiltersRequest")
			if references {
				Meta("openapi:reference:requestBody", `[{"$ref":"#/components/requestBodies/SearchInput","description":""},{"$ref":"#/components/requestBodies/SearchFiltersRequest"}]`)
				Meta("openapi:description:requestBody", "Search filters.")
			}
			Attribute("query", String, func() {
				Example("soup")
			})
			Required("query")
		})

		Service("componentNames", func() {
			explicitReusableComponentMethod("searchRecipes", "/recipes/{widgetID}/search", SearchFilters, references)
			explicitReusableComponentMethod("searchGadgets", "/gadgets/{widgetID}/search", SearchFilters, references)
		})
	}
}

func explicitReusableComponentMethod(name, path string, filters expr.UserType, references bool) {
	Method(name, func() {
		Payload(func() {
			Attribute("widgetID", String, func() {
				Meta("openapi:component:parameter", "WidgetIDParam")
				Example("widget-123")
			})
			Attribute("body", filters)
			Required("widgetID", "body")
		})
		if references {
			Result(func() {
				Attribute("value", String, func() {
					Example("found")
				})
				Attribute("trace", String, "Trace identifier.", func() {
					Example("trace-123")
					Meta("openapi:reference:header", `[{"$ref":"#/components/headers/Trace","description":""}]`)
				})
				Required("value", "trace")
			})
		} else {
			Result(String)
		}
		HTTP(func() {
			POST(path)
			Param("widgetID")
			Body("body")
			Response(StatusOK, func() {
				if references {
					Description("Search results.")
					Meta("openapi:reference:response", `[{"$ref":"#/components/responses/Results","summary":"Search"},{"$ref":"#/components/responses/SearchResults"}]`)
					Header("trace:X-Trace")
					Body("value")
				}
			})
		})
	})
}
