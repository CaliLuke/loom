package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadHeaderStringDefaultValidateDSL = func() {
	Service("ServiceHeaderStringDefaultValidate", func() {
		Method("MethodHeaderStringDefaultValidate", func() {
			Payload(func() {
				Attribute("h", String, func() {
					Default("def")
					Enum("def")
				})
			})
			HTTP(func() {
				GET("/")
				Header("h")
			})
		})
	})
}


