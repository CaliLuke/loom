package testdata

import . "github.com/CaliLuke/loom/dsl"

// UntaggedArraysDSL covers bare collections, page objects, overlapping numeric
// domains, required and optional bodies, and recursively constrained elements.
var UntaggedArraysDSL = func() {
	API("arrayunion", func() {
		Meta("openapi:closed-objects", "true")
	})
	status := Type("Status", String, func() {
		Nullable()
		Enum("open", "closed")
	})
	item := Type("Item", func() {
		Attribute("name", String, func() {
			MinLength(2)
		})
		Attribute("label", String, func() {
			Nullable()
			Enum("open", "closed")
		})
		Required("name")
		Meta("openapi:additionalProperties", "false")
	})
	labels := Type("Labels", MapOf(String, String), func() {
		MinLength(1)
		MaxLength(2)
		Elem(func() {
			MinLength(2)
		})
	})
	tree := Type("Tree", func() {
		Attribute("name", String)
		Attribute("children", MapOf(String, "Tree"))
		Required("name")
	})
	items := Type("Items", ArrayOf(item))
	page := Type("Page", func() {
		Attribute("items", ArrayOf(item))
		Attribute("total", Int)
		Attribute("properties", MapOf(String, Any))
		Attribute("labels", labels)
		Attribute("lookup", MapOf(String, item))
		Attribute("nested", MapOf(String, MapOf(String, ArrayOf(item))))
		Attribute("tree", tree)
		Attribute("nullable", MapOf(String, item), func() {
			Nullable()
			Elem(func() {
				Nullable()
			})
		})
		Required("total")
	})
	result := Type("Result", OneOf(items, page), func() {
		Untagged()
	})
	record := Type("Record", func() {
		Attribute("status", status, func() {
			Enum("open")
		})
		Required("status")
		Meta("openapi:additionalProperties", "false")
	})
	recordOrPage := Type("RecordOrPage", OneOf(record, page), func() {
		Untagged()
	})
	signed := Type("Signed", ArrayOf(Int32))
	unsigned := Type("Unsigned", ArrayOf(UInt32))
	numbers := Type("Numbers", OneOf(signed, unsigned), func() {
		Untagged()
	})
	left := Type("Left", func() {
		Attribute("value", String)
		Attribute("mapping", MapOf(String, String))
	})
	right := Type("Right", func() {
		Attribute("value", String)
		Attribute("mapping", MapOf(String, String))
	})
	overlap := Type("Overlap", OneOf(left, right), func() {
		Untagged()
	})
	positive := Type("ExactPositive", ArrayOf(Float64), func() {
		Elem(func() { Minimum("1e-1000001") })
	})
	text := Type("ExactText", ArrayOf(String))
	exact := Type("ExactUnion", OneOf(positive, text), func() {
		Untagged()
	})
	reverse := Type("ReverseExactUnion", OneOf(text, positive), func() {
		Untagged()
	})
	Service("listing", func() {
		Method("record", func() {
			Result(recordOrPage)
			HTTP(func() { GET("/record") })
		})
		Method("exact", func() {
			Result(exact)
			HTTP(func() { GET("/exact") })
		})
		Method("reverse_exact", func() {
			Result(reverse)
			HTTP(func() { GET("/reverse-exact") })
		})
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
