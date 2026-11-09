package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// ResultViewOpennessDSL exercises authored openness on selected result views.
func ResultViewOpennessDSL() {
	resultViewOpennessDSL(false, "")
}

// ClosedAPIResultViewOpennessDSL distinguishes explicit openness from the API
// default when projecting result declarations.
func ClosedAPIResultViewOpennessDSL() {
	resultViewOpennessDSL(true, "")
}

// ResultDeclarationViewOpennessDSL selects a view on each declaration before
// transport generation, including references that select the same view again.
func ResultDeclarationViewOpennessDSL() {
	resultViewOpennessDSL(false, "tiny")
}

func resultViewOpennessDSL(closedAPI bool, declarationView string) {
	API("view-openness", func() {
		if closedAPI {
			Meta("openapi:closed-objects", "true")
		}
	})
	types := make(map[string]*expr.ResultTypeExpr)
	for _, policy := range []string{"closed", "open", "unset"} {
		types[policy] = ResultType("application/vnd.view-openness."+policy, func() {
			if policy == "closed" {
				Meta("openapi:additionalProperties", "false")
			} else if policy == "open" {
				Meta("openapi:additionalProperties", "true")
			}
			if declarationView != "" {
				Meta(expr.ViewMetaKey, declarationView)
			}
			Attributes(func() {
				Attribute("id", String)
				Attribute("name", String)
				Required("id")
			})
			View("default", func() {
				Attribute("id")
				Attribute("name")
			})
			View("tiny", func() {
				Attribute("id")
			})
		})
	}
	envelope := ResultType("application/vnd.view-openness.envelope", func() {
		Attributes(func() {
			for _, policy := range []string{"closed", "open", "unset"} {
				Attribute(policy, types[policy])
			}
		})
		View("default", func() {
			for _, policy := range []string{"closed", "open", "unset"} {
				Attribute(policy, func() {
					View("tiny")
				})
			}
		})
	})
	Service("view-openness", func() {
		for _, policy := range []string{"closed", "open", "unset"} {
			for _, shape := range []string{"single", "collection"} {
				Method(policy+"_"+shape, func() {
					NoAccessCheck("Public regression specimen")
					result := types[policy]
					if shape == "collection" {
						result = CollectionOf(result)
					}
					Result(result, func() {
						View("tiny")
					})
					HTTP(func() {
						GET("/" + policy + "/" + shape)
					})
				})
			}
		}
		Method("nested", func() {
			NoAccessCheck("Public regression specimen")
			Result(envelope, func() {
				View("default")
			})
			HTTP(func() {
				GET("/nested")
			})
		})
	})
}
