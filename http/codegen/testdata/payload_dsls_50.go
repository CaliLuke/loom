package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadHeaderPrimitiveArrayStringValidateDSL = func() {
	Service("ServiceHeaderPrimitiveArrayStringValidate", func() {
		Method("MethodHeaderPrimitiveArrayStringValidate", func() {
			Payload(ArrayOf(String), func() {
				MinLength(1)
				Elem(func() {
					Pattern("val")
				})
			})
			HTTP(func() {
				GET("/")
				Header("h")
			})
		})
	})
}


