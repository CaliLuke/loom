package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyPrimitiveArrayUserValidateDSL = func() {
	var PayloadType = Type("PayloadType", func() {
		Attribute("a", String, func() {
			Pattern("pattern")
		})
	})
	Service("ServiceBodyPrimitiveArrayUserValidate", func() {
		Method("MethodBodyPrimitiveArrayUserValidate", func() {
			Payload(ArrayOf(PayloadType), func() {
				MinLength(1)
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}


