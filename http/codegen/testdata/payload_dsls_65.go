package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyStringValidateDSL = func() {
	Service("ServiceBodyStringValidate", func() {
		Method("MethodBodyStringValidate", func() {
			Payload(func() {
				Attribute("b", String, func() {
					Pattern("pattern")
				})
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}


