package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadHeaderPrimitiveArrayBoolValidateDSL = func() {
	Service("ServiceHeaderPrimitiveArrayBoolValidate", func() {
		Method("MethodHeaderPrimitiveArrayBoolValidate", func() {
			Payload(ArrayOf(Boolean), func() {
				MinLength(1)
				Elem(func() {
					Enum(true)
				})
			})
			HTTP(func() {
				GET("/")
				Header("h")
			})
		})
	})
}


