package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyQueryPathUserValidateDSL = func() {
	var PayloadType = Type("PayloadType", func() {
		Attribute("a", String, func() {
			Pattern("patterna")
		})
		Attribute("b", String, func() {
			Pattern("patternb")
		})
		Attribute("c", String, func() {
			Pattern("patternc")
		})
		Required("a", "b", "c")
	})
	Service("ServiceBodyQueryPathUserValidate", func() {
		Method("MethodBodyQueryPathUserValidate", func() {
			Payload(PayloadType)
			HTTP(func() {
				POST("/{c}")
				Param("b")
			})
		})
	})
}


