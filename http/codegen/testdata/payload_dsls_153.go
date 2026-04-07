package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadPathCustomFloat32DSL = func() {
	Service("ServicePathCustomFloat32", func() {
		Method("MethodPathCustomFloat32", func() {
			Payload(func() {
				Attribute("p", Float32, func() {
					Meta("struct:field:type", "hide.Float32", "github.com/c2h5oh/hide")
				})
			})
			HTTP(func() {
				GET("/{p}")
			})
		})
	})
}


