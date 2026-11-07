package testdata

import . "github.com/CaliLuke/loom/dsl"

// NullableErrorsDSL exercises nullable error payloads independently of success values.
func NullableErrorsDSL() {
	short := Type("Short", String)
	ordinary := Type("ShortError", String)
	inherited := Type("Inherited", String, func() {
		Nullable()
		Meta("struct:pkg:path", "contracts")
	})
	object := Type("Details", func() {
		Attribute("message", String)
		Required("message")
	})
	identified := Type("Identified", func() {
		ErrorName("name", String)
		Attribute("message", String)
		Required("name", "message")
	})
	API("nullable-errors", func() {})
	Service("nullable", func() {
		Method("scalar", func() {
			Result(ordinary)
			Error("short", short, func() {
				Nullable()
				Description("Short failure")
			})
			HTTP(func() {
				GET("/scalar")
				Response("short", StatusBadRequest)
			})
		})
		Method("object", func() {
			Result(object)
			Error("details", object, func() {
				Nullable()
				Description("Detailed failure")
				Remedy(func() {
					RemedyCode("retry_details")
				})
			})
			HTTP(func() {
				GET("/object")
				Response("details", StatusConflict)
			})
		})
		Method("inherited", func() {
			Error("inherited", inherited)
			HTTP(func() {
				GET("/inherited")
				Response("inherited", StatusBadRequest)
			})
		})
		Method("identified", func() {
			Error("first", identified, func() {
				Nullable()
			})
			Error("second", identified, func() {
				Nullable()
			})
			HTTP(func() {
				GET("/identified")
				Response("first", StatusBadRequest)
				Response("second", StatusConflict)
			})
		})
	})
}
