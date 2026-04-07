package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyPrimitiveArrayUserRequiredDSL = func() {
	var PayloadType = Type("PayloadType", func() {
		Attribute("a", String)
		Required("a")
	})
	Service("ServiceBodyPrimitiveArrayUserRequired", func() {
		Method("MethodBodyPrimitiveArrayUserRequired", func() {
			Payload(ArrayOf(PayloadType))
			HTTP(func() {
				POST("/")
			})
		})
	})
}


