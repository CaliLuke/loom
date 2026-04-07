package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadPathCustomUInt64DSL = func() {
	Service("ServicePathCustomUInt64", func() {
		Method("MethodPathCustomUInt64", func() {
			Payload(func() {
				Attribute("p", UInt64, func() {
					Meta("struct:field:type", "hide.Uint64", "github.com/c2h5oh/hide")
				})
			})
			HTTP(func() {
				GET("/{p}")
			})
		})
	})
}


