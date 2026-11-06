package testdata

import . "github.com/CaliLuke/loom/dsl"

var OpenAPIClosedObjectsDSL = func() {
	var Nested = Type("ClosedObjectsNested", func() {
		Attribute("street", String, func() {
			Example("Main")
		})
		Required("street")
	})

	var OpenExtras = Type("ClosedObjectsOpenExtras", func() {
		Attribute("tag", String, func() {
			Example("t")
		})
		Meta("openapi:additionalProperties", "true")
	})

	var UnionAlpha = Type("ClosedObjectsUnionAlpha", func() {
		Attribute("alpha", String, func() {
			Example("a")
		})
		Required("alpha")
	})

	var UnionBeta = Type("ClosedObjectsUnionBeta", func() {
		Attribute("beta", String, func() {
			Example("b")
		})
		Required("beta")
	})

	var _ = API("test", func() {
		Meta("openapi:closed-objects", "true")
		Server("test", func() {
			Host("localhost", func() {
				URI("https://loom.design")
			})
		})
	})

	Service("closedObjectsService", func() {
		Method("object", func() {
			Payload(func() {
				Attribute("name", String, func() {
					Example("")
				})
				Attribute("address", Nested)
				Attribute("extras", OpenExtras)
				Attribute("open_address", Nested, func() {
					Meta("openapi:additionalProperties", "true")
				})
				Required("name", "address")
			})
			HTTP(func() {
				POST("/object")
			})
		})

		Method("map_object", func() {
			Payload(func() {
				Attribute("labels", MapOf(String, String))
				Required("labels")
			})
			HTTP(func() {
				POST("/map")
			})
		})

		Method("union_object", func() {
			Payload(OneOf(UnionAlpha, UnionBeta))
			HTTP(func() {
				POST("/union")
			})
		})
	})
}
