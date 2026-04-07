package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var StructMetaTypeDSL = func() {
	Service("UsingMetaTypes", func() {
		Method("Method", func() {
			Payload(func() {
				Field(1, "a", Int64, func() {
					Meta("struct:field:type", "flag.ErrorHandling", "flag")
					Default(1)
				})
				Field(2, "b", Int64, func() {
					Meta("struct:field:type", "flag.ErrorHandling", "flag")
					Default(2)
				})
				Field(3, "c", ArrayOf(Int64), func() {
					Elem(func() {
						Meta("struct:field:type", "time.Duration", "time")
					})
				})
				Field(4, "d", Int64, func() {
					Meta("struct:field:type", "flag.ErrorHandling", "flag")
				})
			})
			Result(func() {
				Field(1, "a", Int64, func() {
					Meta("struct:field:type", "flag.ErrorHandling", "flag")
					Default(1)
				})
				Field(2, "b", Int64, func() {
					Meta("struct:field:type", "flag.ErrorHandling", "flag")
					Default(2)
				})
				Field(3, "c", ArrayOf(Int64), func() {
					Elem(func() {
						Meta("struct:field:type", "time.Duration", "time")
					})
				})
				Field(4, "d", Int64, func() {
					Meta("struct:field:type", "flag.ErrorHandling", "flag")
				})
			})
			GRPC(func() {})
		})
	})
}
