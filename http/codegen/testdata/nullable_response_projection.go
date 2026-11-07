package testdata

import . "github.com/CaliLuke/loom/dsl"

// NullableResponseProjectionDSL keeps outer presence separate from selected fields
// and from headers and cookies excluded from a whole response body.
func NullableResponseProjectionDSL() {
	parent := Type("Parent", func() {
		Attribute("message", String)
		Attribute("header", String)
		Attribute("count", Int, func() {
			Meta("struct:field:type", "int8")
		})
		Attribute("session", String, func() {
			Default("fallback")
		})
		Required("message")
	})
	child := Type("Child", func() {
		Attribute("message", String)
		Attribute("detail", Int, func() {
			Nullable()
		})
		Attribute("header", String)
		Required("detail")
	})
	object := Type("ObjectParent", func() {
		Attribute("detail", parent)
		Attribute("header", String)
		Required("detail")
	})
	formattedChildMethod := nullableFormattedChildMethod()
	Service("projection", func() {
		formattedChildMethod()
		for _, mode := range []string{"selected", "whole", "required", "child", "object"} {
			Method(mode, func() {
				typ := parent
				if mode == "child" {
					typ = child
				} else if mode == "object" {
					typ = object
				}
				Result(typ, func() {
					Nullable()
				})
				Error(mode, typ, func() {
					Nullable()
				})
				mapping := func() {
					Headers(func() {
						Header("header:X-Info")
						if mode == "required" {
							Required("header")
						}
					})
					if mode != "child" && mode != "object" {
						Header("count:X-Count")
						Cookie("session:Session")
					}
					if mode == "child" || mode == "object" {
						Body("detail")
					} else if mode == "selected" {
						Body("message")
					}
				}
				HTTP(func() {
					GET("/" + mode)
					Response(StatusOK, mapping)
					Response(mode, StatusBadRequest, mapping)
				})
			})
		}
	})
}

// nullableFormattedChildMethod declares its type before service evaluation and
// defers only the method, keeping fixture declaration order stable.
func nullableFormattedChildMethod() func() {
	formattedChild := Type("FormattedChild", func() {
		Attribute("detail", Int, func() {
			Nullable()
		})
		Required("detail")
	})
	return func() {
		Method("formatted_child", func() {
			Error("formatted_child", formattedChild, func() {
				Nullable()
			})
			HTTP(func() {
				GET("/formatted_child")
				Response("formatted_child", StatusBadRequest, func() {
					Body("detail")
				})
			})
		})
	}
}
