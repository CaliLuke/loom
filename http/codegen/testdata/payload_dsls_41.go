package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadHeaderStringValidateDSL = func() {
	Service("ServiceHeaderStringValidate", func() {
		Method("MethodHeaderStringValidate", func() {
			Payload(func() {
				Attribute("h", String, func() {
					Pattern("header")
				})
			})
			HTTP(func() {
				GET("/")
				Header("h")
			})
		})
	})
}


