package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadPathCustomInt64DSL = func() {
	Service("ServicePathCustomInt64", func() {
		Method("MethodPathCustomInt64", func() {
			Payload(func() {
				Attribute("p", Int64, func() {
					Meta("struct:field:type", "hide.Int64", "github.com/c2h5oh/hide")
				})
			})
			HTTP(func() {
				GET("/{p}")
			})
		})
	})
}


