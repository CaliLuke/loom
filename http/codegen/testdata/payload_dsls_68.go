package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyNestedUserDSL = func() {
	var NestedType = Type("NestedType", func() {
		Attribute("a", String)
		Attribute("b", String)
		Required("a")
	})
	var PayloadType = Type("PayloadType", func() {
		Attribute("data", NestedType)
	})
	Service("ServiceBodyUser", func() {
		Method("MethodBodyUser", func() {
			Payload(PayloadType)
			HTTP(func() {
				POST("/")
				Body("data")
			})
		})
	})
}


