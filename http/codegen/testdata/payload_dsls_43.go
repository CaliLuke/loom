package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadHeaderArrayStringValidateDSL = func() {
	Service("ServiceHeaderArrayStringValidate", func() {
		Method("MethodHeaderArrayStringValidate", func() {
			Payload(func() {
				Attribute("h", ArrayOf(String, func() {
					Enum("val")
				}))
			})
			HTTP(func() {
				GET("/")
				Header("h")
			})
		})
	})
}


