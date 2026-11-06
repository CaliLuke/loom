package codegen

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
)

func TestParameterNativeCarriersGenerated(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		record := Type("Record", func() {
			Attribute("name", String)
			Attribute("count", Int)
			Required("name", "count")
		})
		result := ResultType("Viewed", func() {
			Attribute("range", record)
			Required("range")
			View("default", func() { Attribute("range") })
			View("tiny", func() { Attribute("range") })
		})
		Service("carriers", func() {
			Method("inline", func() {
				Payload(func() {
					Attribute("value", func() {
						Attribute("count", Int, func() { Default(7) })
						Attribute("name", String)
					})
				})
				HTTP(func() {
					GET("/inline")
					Param("value", func() { Meta("openapi:style", "deepObject") })
					Response(StatusNoContent)
				})
			})
			Method("view", func() {
				Result(result)
				HTTP(func() {
					GET("/view")
					Response(StatusOK, func() {
						Header("range:X-Range", func() { Meta("openapi:explode", "true") })
						Body(Empty)
					})
				})
			})
			Method("custom", func() {
				Payload(func() {
					Attribute("id", String, func() {
						Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
						MinLength(1)
					})
					Required("id")
				})
				HTTP(func() {
					GET("/custom/{id}")
					Param("id", func() { Meta("openapi:style", "matrix") })
					Response(StatusNoContent)
				})
			})
			Method("names", func() {
				Payload(func() {
					Attribute("parameterValue0", String)
					Attribute("parameterSegment1", String)
					Required("parameterValue0", "parameterSegment1")
				})
				HTTP(func() {
					GET("/names/{parameterValue0}/{parameterSegment1}")
					Param("parameterValue0", func() { Meta("openapi:style", "matrix") })
					Param("parameterSegment1", func() { Meta("openapi:style", "label") })
					Response(StatusNoContent)
				})
			})
		})
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/nativecarriers", root)
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}
