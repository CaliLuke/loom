package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyQueryUserUnionValidateDSL = func() {
	var Union = Type("Union", func() {
		OneOf("Values", func() {
			Attribute("String", String)
			Attribute("Int", Int)
		})
	})
	var PayloadType = Type("PayloadType", func() {
		Attribute("a", Union)
		Attribute("b", String, func() {
			Pattern("patternb")
		})
		Required("a", "b")
	})
	Service("ServiceBodyQueryUserUnionValidate", func() {
		Method("MethodBodyQueryUserUnionValidate", func() {
			Payload(PayloadType)
			HTTP(func() {
				POST("/")
				Param("b")
			})
		})
	})
}


