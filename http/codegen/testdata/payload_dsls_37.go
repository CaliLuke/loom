package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadPathPrimitiveBoolValidateDSL = func() {
	Service("ServicePathPrimitiveBoolValidate", func() {
		Method("MethodPathPrimitiveBoolValidate", func() {
			Payload(Boolean, func() {
				Enum(true)
			})
			HTTP(func() {
				GET("/{p}")
			})
		})
	})
}


