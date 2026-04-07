package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var StructFieldNameMetaTypeDSL = func() {
	Service("UsingMetaTypes", func() {
		Method("Method", func() {
			Payload(func() {
				Field(1, "a", Int64, func() {
					Meta("struct:field:name", "Foo")
					Default(1)
				})
				Field(2, "b", ArrayOf(Int64), func() {
					Meta("struct:field:name", "Bar")
				})
				Required("b")
			})
			Result(func() {
				Field(1, "a", Int64, func() {
					Meta("struct:field:name", "Foo")
					Default(1)
				})
				Field(2, "b", ArrayOf(Int64), func() {
					Meta("struct:field:name", "Bar")
				})
				Required("b")
			})
			GRPC(func() {})
		})
	})
}
