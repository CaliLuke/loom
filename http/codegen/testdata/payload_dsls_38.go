package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadPathPrimitiveArrayStringValidateDSL = func() {
	Service("ServicePathPrimitiveArrayStringValidate", func() {
		Method("MethodPathPrimitiveArrayStringValidate", func() {
			Payload(ArrayOf(String), func() {
				MinLength(1)
				Elem(func() {
					Enum("val")
				})
			})
			HTTP(func() {
				GET("/{p}")
			})
		})
	})
}


