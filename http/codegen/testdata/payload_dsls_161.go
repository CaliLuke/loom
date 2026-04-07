package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyCustomNameDSL = func() {
	Service("ServiceBodyCustomName", func() {
		Method("MethodBodyCustomName", func() {
			Payload(func() {
				Attribute("b", String, func() {
					Meta("struct:field:name", "Body")
				})
			})
			HTTP(func() {
				GET("/")
			})
		})
	})
}


