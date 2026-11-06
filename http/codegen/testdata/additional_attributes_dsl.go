package testdata

import . "github.com/CaliLuke/loom/dsl"

// AdditionalAttributesTypedDSL declares objects that combine declared members
// with typed additional members, including required members that only the
// typed policy describes, nesting, Extend, user types that only additional
// members reach, validations that only additional members carry, recursion
// through additional members and members bound to path parameters.
var AdditionalAttributesTypedDSL = func() {
	var Labels = Type("Labels", func() {
		Attribute("id", String)
		AdditionalAttributes(Int, func() {
			Minimum(0)
		})
		Required("id", "count")
	})

	var Tagged = Type("Tagged", func() {
		Extend(Labels)
		Attribute("total", Int)
	})

	var Item = Type("Item", func() {
		Attribute("sku", String, func() {
			MinLength(1)
		})
		Required("sku")
	})

	var Bag = Type("Bag", func() {
		AdditionalAttributes(Item)
		Required("must")
	})

	var Lists = Type("Lists", func() {
		Attribute("kind", String)
		AdditionalAttributes(ArrayOf(String, func() {
			MinLength(1)
		}))
	})

	var Node = Type("Node", func() {
		Attribute("value", Int, func() {
			Minimum(0)
		})
		AdditionalAttributes("Node")
	})

	var Opt = Type("Opt", func() {
		Attribute("id", Int)
		Attribute("name", String)
		AdditionalAttributes(Int)
	})

	var _ = API("additional", func() {
		Server("additional", func() {
			Host("localhost", func() {
				URI("http://localhost:8080")
			})
		})
	})

	Service("records", func() {
		Method("put", func() {
			Payload(func() {
				Attribute("labels", Labels)
				Attribute("tagged", Tagged)
				Required("labels")
			})
			Result(Labels)
			HTTP(func() {
				POST("/records")
			})
		})
		Method("bag", func() {
			Payload(Bag)
			Result(Bag)
			HTTP(func() {
				POST("/bag")
			})
		})
		Method("nest", func() {
			Payload(func() {
				Attribute("lists", Lists)
				Attribute("node", Node)
			})
			Result(func() {
				Attribute("lists", Lists)
				Attribute("node", Node)
			})
			HTTP(func() {
				POST("/nest")
			})
		})
		Method("opt", func() {
			Payload(Opt)
			Result(Opt)
			HTTP(func() {
				POST("/o/{id}")
				Response(StatusOK, func() {
					Header("id")
				})
			})
		})
		Method("bound", func() {
			Payload(Labels)
			HTTP(func() {
				POST("/p/{id}")
			})
		})
	})
}
