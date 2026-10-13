package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// ViewedUnionOptions selects bounded package-membership controls.
type ViewedUnionOptions struct {
	// JSONRPC selects the JSON-RPC transport.
	JSONRPC bool
	// IncludeShape includes the nested union in the selected default view.
	IncludeShape bool
	// ReuseRequest also uses the union in a request body.
	ReuseRequest bool
	// Untagged uses named object variants with validation requirements.
	Untagged bool
	// MapItems wraps nested result values in a map rather than an array.
	MapItems bool
}

// ViewedOmittedUnionDSL reproduces a nested union excluded by response views.
func ViewedOmittedUnionDSL() {
	ViewedUnionDSL(ViewedUnionOptions{})()
}

// ViewedOmittedUnionJSONRPCDSL covers the same projection through JSON-RPC.
func ViewedOmittedUnionJSONRPCDSL() {
	ViewedUnionDSL(ViewedUnionOptions{JSONRPC: true})()
}

// ViewedUnionDSL constructs a reusable selected-view declaration fixture.
func ViewedUnionDSL(options ViewedUnionOptions) func() {
	return func() {
		shape := viewedUnionShape(options.Untagged)
		item := viewedUnionItem(shape, options.IncludeShape)
		wrap := viewedUnionWrap(item, options.MapItems)

		Service("svc", func() {
			if options.JSONRPC {
				JSONRPC(func() {
					POST("/rpc")
				})
			}
			Method("Get", func() {
				Result(wrap)
				if options.JSONRPC {
					JSONRPC(func() {
					})
				} else {
					HTTP(func() {
						GET("/get")
					})
				}
			})
			if options.ReuseRequest {
				Method("Reuse", func() {
					Payload(shape)
					if options.JSONRPC {
						JSONRPC(func() {
						})
					} else {
						HTTP(func() {
							POST("/reuse")
						})
					}
				})
			}
		})
	}
}

func viewedUnionShape(untagged bool) expr.UserType {
	var shape expr.UserType
	if untagged {
		circle := Type("Circle", func() {
			Attribute("radius", Int, func() {
				Minimum(1)
			})
		})
		name := Type("Name", func() {
			Attribute("label", String)
		})
		shape = Type("Shape", OneOf(circle, name), func() {
			Untagged()
		})
	} else {
		shape = Type("Shape", func() {
			OneOf("kind", func() {
				Field(1, "circle", Int)
				Field(2, "name", String)
			})
		})
	}
	return shape
}

func viewedUnionItem(shape expr.UserType, includeShape bool) *expr.ResultTypeExpr {
	return ResultType("application/vnd.item", func() {
		TypeName("Item")
		Attributes(func() {
			Field(1, "id", String)
			Field(2, "shape", shape)
			Required("id")
		})
		View("default", func() {
			Attribute("id")
			if includeShape {
				Attribute("shape")
			}
		})
		View("shaped", func() {
			Attribute("id")
			Attribute("shape")
		})
	})
}

func viewedUnionWrap(item *expr.ResultTypeExpr, mapItems bool) *expr.ResultTypeExpr {
	return ResultType("application/vnd.wrap", func() {
		TypeName("Wrap")
		Attributes(func() {
			Field(1, "id", String)
			if mapItems {
				Field(2, "items", MapOf(String, item))
			} else {
				Field(2, "items", ArrayOf(item))
			}
			Required("id")
		})
		View("default", func() {
			Attribute("id")
			Attribute("items")
		})
		View("tiny", func() {
			Attribute("id")
		})
	})
}
