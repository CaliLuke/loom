package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.

var PayloadPathCustomUIntDSL = func() {
	Service("ServicePathCustomUInt", func() {
		Method("MethodPathCustomUInt", func() {
			Payload(func() {
				Attribute("p", UInt, func() {
					Meta("struct:field:type", "custom.Uint", "github.com/CaliLuke/loom/http/codegen/testdata/custom")
				})
			})
			HTTP(func() {
				GET("/{p}")
			})
		})
	})
}
