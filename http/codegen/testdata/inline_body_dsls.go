package testdata

import . "github.com/CaliLuke/loom/dsl"

// InlineBodySelectionDSL defines inline request and response Body DSLs that
// select payload and result attributes, including user-type, collection and
// inline object attributes, while other attributes map to params and headers.
var InlineBodySelectionDSL = func() {
	item := Type("Item", func() {
		Attribute("x", String)
	})
	empty := Type("EmptyRecord", func() {
		Meta("openapi:additionalProperties", "false")
	})
	failure := Type("EmptyFailure", func() {
		ErrorName("name", String)
		Attribute("body", empty)
		Required("name", "body")
	})
	Service("InlineBody", func() {
		Method("empty_selected", func() {
			Result(func() {
				Attribute("body", empty)
				Attribute("h", String)
				Required("body", "h")
			})
			HTTP(func() {
				GET("/empty_selected")
				Response(StatusOK, func() {
					Header("h:X-Label")
					Body("body")
					OpenAPIBody(empty)
				})
			})
		})
		Method("empty_type", func() {
			Result(empty)
			HTTP(func() {
				GET("/empty_type")
				Response(StatusOK, func() {
					Body(empty)
				})
			})
		})
		Method("empty_error", func() {
			Error("empty", failure)
			HTTP(func() {
				GET("/empty_error")
				Response("empty", StatusBadRequest, func() {
					Header("name:X-Error")
					Body("body")
					OpenAPIBody(empty)
				})
			})
		})
		Method("primitive", func() {
			Payload(func() {
				Attribute("name", String)
				Attribute("q", String)
				Required("name")
			})
			Result(func() {
				Attribute("name", String)
				Attribute("h", String)
			})
			HTTP(func() {
				POST("/primitive")
				Param("q")
				Body(func() {
					Attribute("name")
					Required("name")
				})
				Response(StatusOK, func() {
					Header("h")
					Body(func() {
						Attribute("name")
					})
				})
			})
		})
		Method("single", func() {
			Payload(func() {
				Attribute("item", item)
				Attribute("q", String)
			})
			Result(func() {
				Attribute("item", item)
				Attribute("h", String)
			})
			HTTP(func() {
				POST("/single")
				Param("q")
				Body(func() {
					Attribute("item")
				})
				Response(StatusOK, func() {
					Header("h")
					Body(func() {
						Attribute("item")
					})
				})
			})
		})
		Method("required", func() {
			Payload(func() {
				Attribute("item", item)
				Attribute("q", String)
				Required("item", "q")
			})
			Result(func() {
				Attribute("item", item)
				Attribute("h", String)
				Required("item", "h")
			})
			HTTP(func() {
				POST("/required")
				Param("q")
				Body(func() {
					Attribute("item")
					Required("item")
				})
				Response(StatusOK, func() {
					Header("h")
					Body(func() {
						Attribute("item")
						Required("item")
					})
				})
			})
		})
		Method("several", func() {
			Payload(func() {
				Attribute("item", item)
				Attribute("items", ArrayOf(item))
				Attribute("name", String)
				Attribute("count", Int)
				Attribute("q", String)
				Required("name")
			})
			Result(func() {
				Attribute("item", item)
				Attribute("items", ArrayOf(item))
				Attribute("name", String)
				Attribute("h", String)
				Required("name")
			})
			HTTP(func() {
				POST("/several")
				Param("q")
				Body(func() {
					Attribute("item")
					Attribute("items")
					Attribute("name")
					Attribute("count")
					Required("name")
				})
				Response(StatusOK, func() {
					Header("h")
					Body(func() {
						Attribute("item")
						Attribute("items")
						Attribute("name")
					})
				})
			})
		})
		Method("inline_attribute", func() {
			Payload(func() {
				Attribute("inline", func() {
					Attribute("y", String)
				})
				Attribute("q", String)
			})
			Result(func() {
				Attribute("inline", func() {
					Attribute("y", String)
				})
				Attribute("h", String)
			})
			HTTP(func() {
				POST("/inline_attribute")
				Param("q")
				Body("inline")
				Response(StatusOK, func() {
					Header("h")
					Body("inline")
				})
			})
		})
	})
}

// InlineBodySharedMethodNameDSL defines two services whose methods share a name
// and both select an inline request body.
var InlineBodySharedMethodNameDSL = func() {
	for _, svc := range []string{"First", "Second"} {
		Service(svc, func() {
			Method("do", func() {
				Payload(func() {
					Attribute("name", String)
					Attribute("q", String)
				})
				HTTP(func() {
					POST("/" + svc + "/do")
					Param("q")
					Body(func() {
						Attribute("name")
					})
				})
			})
		})
	}
}
