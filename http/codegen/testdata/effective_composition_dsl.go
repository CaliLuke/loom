package testdata

import . "github.com/CaliLuke/loom/dsl"

// EffectiveCompositionDSL exercises field refinement through both HTTP and
// gRPC: inherited requiredness, numeric conjunction and an authored default.
var EffectiveCompositionDSL = func() {
	base := Type("CompositionBase", func() {
		Field(1, "count", Int, func() {
			Minimum(0)
		})
		Field(2, "label", String, func() {
			MinLength(1)
		})
		Field(3, "retries", Int, func() {
			Minimum(0)
			Default(2)
		})
		Required("count", "label")
	})
	derived := Type("CompositionDerived", func() {
		Extend(base)
		Field(1, "count", Int, func() {
			Maximum(100)
		})
		Field(3, "retries", Int, func() {
			Maximum(10)
			Default(5)
		})
	})
	Service("composition", func() {
		Method("read", func() {
			Payload(derived)
			Result(derived)
			HTTP(func() {
				POST("/read")
			})
			GRPC(func() {

			})
		})
	})
}
