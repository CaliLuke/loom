package testdata

import . "github.com/CaliLuke/loom/dsl"

// UntaggedArraysDSL covers bare collections, page objects, overlapping numeric
// domains, required and optional bodies, and recursively constrained elements.
var UntaggedArraysDSL = func() {
	API("arrayunion", func() {
		Meta("openapi:closed-objects", "true")
	})
	item := Type("Item", func() {
		Attribute("name", String, func() {
			MinLength(2)
		})
		Attribute("label", String, func() {
			Nullable()
		})
		Required("name")
		Meta("openapi:additionalProperties", "false")
	})
	items := Type("Items", ArrayOf(item))
	page := Type("Page", func() {
		Attribute("items", ArrayOf(item))
		Attribute("total", Int)
		Required("total")
	})
	result := Type("Result", OneOf(items, page), func() {
		Untagged()
	})
	signed := Type("Signed", ArrayOf(Int32))
	unsigned := Type("Unsigned", ArrayOf(UInt32))
	numbers := Type("Numbers", OneOf(signed, unsigned), func() {
		Untagged()
	})
	left := Type("Left", func() {
		Attribute("value", String)
	})
	right := Type("Right", func() {
		Attribute("value", String)
	})
	overlap := Type("Overlap", OneOf(left, right), func() {
		Untagged()
	})
	Service("listing", func() {
		Method("echo", func() {
			NoSecurity()
			Payload(result)
			Result(result)
			HTTP(func() {
				POST("/echo")
			})
		})
		Method("optional", func() {
			NoSecurity()
			Payload(func() {
				Attribute("body", result)
			})
			HTTP(func() {
				POST("/optional")
				Body("body")
			})
		})
		Method("ambiguous", func() {
			NoSecurity()
			Result(overlap)
			HTTP(func() {
				GET("/overlap")
			})
		})
		Method("numeric", func() {
			NoSecurity()
			Result(numbers)
			HTTP(func() {
				GET("/numbers")
			})
		})
	})
}
