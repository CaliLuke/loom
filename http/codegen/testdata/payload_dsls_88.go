package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyPrimitiveStringValidateDSL = func() {
	Service("ServiceBodyPrimitiveStringValidate", func() {
		Method("MethodBodyPrimitiveStringValidate", func() {
			Payload(String, func() {
				Enum("val")
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}


