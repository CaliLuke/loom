package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyUserValidateDSL = func() {
	var PayloadType = Type("PayloadType", func() {
		Attribute("a", String, func() {
			Pattern("apattern")
		})
	})
	Service("ServiceBodyUserValidate", func() {
		Method("MethodBodyUserValidate", func() {
			Payload(PayloadType)
			HTTP(func() {
				POST("/")
				Body("a")
			})
		})
	})
}


