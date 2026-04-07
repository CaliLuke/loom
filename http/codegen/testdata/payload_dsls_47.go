package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadHeaderArrayIntValidateDSL = func() {
	Service("ServiceHeaderArrayIntValidate", func() {
		Method("MethodHeaderArrayIntValidate", func() {
			Payload(func() {
				Attribute("h", ArrayOf(Int, func() {
					Enum(1, 2)
				}))
			})
			HTTP(func() {
				GET("/")
				Header("h")
			})
		})
	})
}


