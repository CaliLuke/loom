package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadExtendBodyPrimitiveFieldStringDSL = func() {
	var Ext = Type("Ext", func() {
		Attribute("b", String)
	})
	var PayloadType = Type("PayloadType", func() {
		Extend(Ext)
	})
	Service("ServiceBodyPrimitiveArrayUser", func() {
		Method("MethodBodyPrimitiveArrayUser", func() {
			Payload(PayloadType)
			HTTP(func() {
				POST("/")
				Body("b")
			})
		})
	})
}


