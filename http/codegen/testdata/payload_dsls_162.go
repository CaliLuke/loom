package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadPathCustomNameDSL = func() {
	Service("ServicePathCustomName", func() {
		Method("MethodPathCustomName", func() {
			Payload(func() {
				Attribute("p", String, func() {
					Meta("struct:field:name", "Path")
				})
				Required("p")
			})
			HTTP(func() {
				GET("/{p}")
			})
		})
	})
}


