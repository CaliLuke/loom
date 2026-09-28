package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// UnionCollectionsDSL exercises anonymous and named collections of sum types
// as complete HTTP request and response bodies, including nested collections.
var UnionCollectionsDSL = func() {
	choices := Type("Choices", ArrayOf(OneOf(Float32, String)))
	Service("choices", func() {
		Method("anonymous", func() {
			Payload(ArrayOf(OneOf(Float32, String)))
			Result(ArrayOf(OneOf(Float32, String)))
			HTTP(func() {
				POST("/anonymous")
			})
		})
		Method("named", func() {
			Payload(choices)
			Result(choices)
			HTTP(func() {
				POST("/named")
			})
		})
		Method("nested", func() {
			Payload(ArrayOf(ArrayOf(OneOf(Float32, String))))
			Result(ArrayOf(ArrayOf(OneOf(Float32, String))))
			HTTP(func() {
				POST("/nested")
			})
		})
		Method("mapped", func() {
			Payload(MapOf(String, OneOf(Float32, String)))
			Result(MapOf(String, OneOf(Float32, String)))
			HTTP(func() {
				POST("/mapped")
			})
		})
	})
}
