package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadPathCustomUInt32DSL = func() {
	Service("ServicePathCustomUInt32", func() {
		Method("MethodPathCustomUInt32", func() {
			Payload(func() {
				Attribute("p", UInt32, func() {
					Meta("struct:field:type", "hide.Uint32", "github.com/c2h5oh/hide")
				})
			})
			HTTP(func() {
				GET("/{p}")
			})
		})
	})
}


