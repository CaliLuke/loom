package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyMapUserValidateDSL = func() {
	var PayloadType = Type("PayloadType", func() {
		Attribute("a", String, func() {
			Pattern("apattern")
		})
	})
	Service("ServiceBodyMapUserValidate", func() {
		Method("MethodBodyMapUserValidate", func() {
			Payload(func() {
				Attribute("b", MapOf(String, PayloadType), func() {
					Key(func() {
						MinLength(2)
					})
				})
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}


