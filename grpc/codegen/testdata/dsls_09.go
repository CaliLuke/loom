package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// InlineObjectFieldsDSL declares fields whose types are anonymous inline
// objects: an object that holds only an anonymous OneOf with an inline object
// branch, nested inline objects, and inline objects as array elements and map
// values, nested collections of inline objects, an inline array element whose
// nested object has a defaulted field, a field whose generated message name
// coincides with the one of a nested object, and a user type whose name is
// the generated message name of a nested object. Protocol buffer fields cannot have anonymous message types,
// so each inline object needs a distinct generated message.
var InlineObjectFieldsDSL = func() {
	var Named = Type("UnaryRequest_outer_inner", func() {
		Field(1, "y", String)
	})
	var Wrapper = Type("Wrapper", func() {
		Field(1, "id", String)
		Field(5, "name", func() {
			OneOf("choice", func() {
				Field(1, "text", String)
				Field(2, "num", Int)
				Field(3, "pair", func() {
					Field(1, "key", String)
					Field(2, "value", String)
				})
			})
			Required("choice")
		})
		Field(6, "outer", func() {
			Field(1, "label", String)
			Field(2, "inner", func() {
				Field(1, "depth", Int)
				Required("depth")
			})
		})
		Field(7, "items", ArrayOf(&expr.Object{}, func() {
			Field(1, "key", String)
		}))
		Field(8, "index", MapOf(String, &expr.Object{}, func() {
			Elem(func() {
				Field(1, "count", Int)
			})
		}))
		Field(9, "outer_inner", func() {
			Field(1, "q", String)
		})
		Field(10, "named", Named)
		Field(11, "grid", ArrayOf(ArrayOf(&expr.Object{}, func() {
			Field(1, "cell", String)
		})))
		Field(12, "groups", MapOf(String, ArrayOf(&expr.Object{}, func() {
			Field(1, "member", String)
		})))
		Field(13, "rows", ArrayOf(&expr.Object{}, func() {
			Field(1, "m", func() {
				Field(1, "z", Int, func() {
					Default(3)
				})
			})
		}))
		Required("outer")
	})
	Service("InlineObjectFields", func() {
		Method("Unary", func() {
			Payload(Wrapper)
			Result(Wrapper)
			GRPC(func() {})
		})
	})
}
