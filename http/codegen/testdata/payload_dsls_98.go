package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyPrimitiveFieldArrayUserValidateDSL = func() {
	var PayloadType = Type("PayloadType", func() {
		Attribute("a", ArrayOf(String), func() {
			MinLength(1)
			Elem(func() {
				Pattern("pattern")
			})
		})
		Required("a")
	})
	Service("ServiceBodyPrimitiveArrayUserValidate", func() {
		Method("MethodBodyPrimitiveArrayUserValidate", func() {
			Payload(PayloadType)
			HTTP(func() {
				POST("/")
				Body("a")
			})
		})
	})
}


