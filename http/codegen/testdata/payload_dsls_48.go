package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadHeaderPrimitiveStringValidateDSL = func() {
	Service("ServiceHeaderPrimitiveStringValidate", func() {
		Method("MethodHeaderPrimitiveStringValidate", func() {
			Payload(String, func() {
				Enum("val")
			})
			HTTP(func() {
				GET("/")
				Header("h")
			})
		})
	})
}


