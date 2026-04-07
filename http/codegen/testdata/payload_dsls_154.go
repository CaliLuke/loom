package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadPathCustomFloat64DSL = func() {
	Service("ServicePathCustomFloat64", func() {
		Method("MethodPathCustomFloat64", func() {
			Payload(func() {
				Attribute("p", Float64, func() {
					Meta("struct:field:type", "hide.Float64", "github.com/c2h5oh/hide")
				})
			})
			HTTP(func() {
				GET("/{p}")
			})
		})
	})
}


