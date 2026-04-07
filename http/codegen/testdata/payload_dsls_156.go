package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadPathCustomInt32DSL = func() {
	Service("ServicePathCustomInt32", func() {
		Method("MethodPathCustomInt32", func() {
			Payload(func() {
				Attribute("p", Int32, func() {
					Meta("struct:field:type", "hide.Int32", "github.com/c2h5oh/hide")
				})
			})
			HTTP(func() {
				GET("/{p}")
			})
		})
	})
}


