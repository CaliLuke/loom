package testdata

import . "github.com/CaliLuke/loom/dsl"

// EffectiveCompositionDSL exercises field refinement through both HTTP and
// gRPC: inherited requiredness, numeric conjunction and an authored default.
var EffectiveCompositionDSL = effectiveCompositionDSL(false)

// ReferenceCompositionDSL preserves the named contract contributor while
// exercising the same runtime intersection as EffectiveCompositionDSL.
var ReferenceCompositionDSL = effectiveCompositionDSL(true)

func effectiveCompositionDSL(reference bool) func() {
	return func() {
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
			if reference {
				Field(4, "digest", Bytes, func() {
					MinLength(2)
				})
			}
			Required("count", "label")
		})
		derived := Type("CompositionDerived", func() {
			if reference {
				Meta("openapi:allOf:reference", "true")
			}
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
}
