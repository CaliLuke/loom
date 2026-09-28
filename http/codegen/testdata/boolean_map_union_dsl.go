package testdata

import . "github.com/CaliLuke/loom/dsl"

// BooleanMapUnionDSL exercises boolean-keyed maps at union, body, and nested
// nullable boundaries, using required and optional request bodies.
var BooleanMapUnionDSL = func() {
	API("unionkeys", func() {})
	key := Type("FlagKey", Boolean)
	nested := Type("Nested", func() {
		Attribute("flags", MapOf(key, Boolean), func() {
			Nullable()
		})
		Attribute("marker", String)
		Required("marker")
	})
	other := Type("Other", func() {
		Attribute("text", String)
		Required("text")
	})
	wrapped := Type("Wrapped", func() {
		Attribute("payload", nested)
		Required("payload")
	})
	choice := func() {
		OneOf("choice", func() {
			Attribute("flags", MapOf(Boolean, String))
			Attribute("nested", nested)
			Attribute("text", String)
		})
	}
	tagged := Type("Tagged", func() {
		choice()
	})
	optional := Type("Optional", choice)
	required := Type("Required", func() {
		choice()
		Required("choice")
	})
	untagged := Type("Untagged", func() {
		Attribute("choice", OneOf(wrapped, other), func() {
			Untagged()
		})
	})
	Service("probe", func() {
		Method("required", func() {
			Payload(required)
			HTTP(func() {
				POST("/required")
				Body("choice")
			})
		})
		Method("tagged", func() {
			Payload(tagged)
			Result(tagged)
			HTTP(func() {
				POST("/tagged")
				Body("choice")
				Response(StatusOK, func() {
					Body("choice")
				})
			})
		})
		Method("untagged", func() {
			Payload(untagged)
			Result(untagged)
			HTTP(func() {
				POST("/untagged")
				Body("choice")
				Response(StatusOK, func() {
					Body("choice")
				})
			})
		})
		Method("optional", func() {
			Payload(optional)
			HTTP(func() {
				POST("/optional")
				Body("choice")
			})
		})
	})
}
