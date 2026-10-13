package generator

import (
	"testing"

	"github.com/CaliLuke/loom/dsl"
)

// TestSelectedViewCarrierCompile compiles and exercises each authored nullable carrier.
func TestSelectedViewCarrierCompile(t *testing.T) {
	for _, carrier := range []string{"ordinary", "occurrence", "intrinsic", "root-occurrence"} {
		t.Run(carrier, func(t *testing.T) {
			runDesignHarness(t, "example.com/viewcarrier", func() {
				selectedViewCarrierDSL(carrier)
			}, selectedViewCarrierHarness(carrier))
		})
	}
}

func selectedViewCarrierDSL(carrier string) {
	dsl.API("viewcarrier", func() {
		dsl.JSONRPC(func() {
		})
	})
	child := dsl.ResultType("application/vnd.child", "Child", func() {
		dsl.Attributes(func() {
			if carrier == "intrinsic" {
				dsl.Nullable()
			}
			dsl.Attribute("text", dsl.String, func() {
				dsl.MinLength(1)
			})
			dsl.Attribute("label", dsl.String)
			dsl.Required("text")
		})
		dsl.View("default", func() {
			dsl.Attribute("text")
			dsl.Attribute("label")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("label")
		})
	})
	parent := dsl.ResultType("application/vnd.parent", "Parent", func() {
		dsl.Attribute("optional", child, func() {
			if carrier == "occurrence" {
				dsl.Nullable()
			}
		})
		dsl.Attribute("required", child, func() {
			if carrier == "occurrence" {
				dsl.Nullable()
			}
		})
		dsl.Attribute("label", dsl.String)
		dsl.Required("required")
		dsl.View("default", func() {
			dsl.Attribute("optional")
			dsl.Attribute("required")
			dsl.Attribute("label")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("label")
		})
		dsl.View("nested", func() {
			dsl.Attribute("optional", func() {
				dsl.View("tiny")
			})
			dsl.Attribute("required", func() {
				dsl.View("tiny")
			})
		})
	})
	children := dsl.CollectionOf(child)
	parents := dsl.CollectionOf(parent)
	for _, name := range []string{"views", "viewsrpc"} {
		dsl.Service(name, func() {
			if name == "viewsrpc" {
				dsl.JSONRPC(func() {
					dsl.POST("/rpc")
				})
			}
			for _, method := range []string{"show", "root", "children", "parents"} {
				dsl.Method(method, func() {
					switch method {
					case "show":
						dsl.Result(parent)
					case "root":
						dsl.Result(child, func() {
							if carrier == "root-occurrence" {
								dsl.Nullable()
							}
						})
					case "children":
						dsl.Result(children, func() {
							if carrier == "root-occurrence" {
								dsl.Nullable()
							}
						})
					case "parents":
						dsl.Result(parents)
					}
					if name == "viewsrpc" {
						dsl.JSONRPC(func() {
						})
					} else {
						dsl.HTTP(func() {
							dsl.GET("/" + method)
						})
					}
				})
			}
		})
	}
}
