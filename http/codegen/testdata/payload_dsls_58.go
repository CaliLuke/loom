package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadCookiePrimitiveBoolValidateDSL = func() {
	Service("ServiceCookiePrimitiveBoolValidate", func() {
		Method("MethodCookiePrimitiveBoolValidate", func() {
			Payload(Boolean, func() {
				Enum(true)
			})
			HTTP(func() {
				GET("/")
				Cookie("c")
			})
		})
	})
}


