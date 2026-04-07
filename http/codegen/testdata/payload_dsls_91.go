package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyPrimitiveArrayBoolValidateDSL = func() {
	Service("ServiceBodyPrimitiveArrayBoolValidate", func() {
		Method("MethodBodyPrimitiveArrayBoolValidate", func() {
			Payload(ArrayOf(Boolean), func() {
				MinLength(1)
				Elem(func() {
					Enum(true)
				})
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}


