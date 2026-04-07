package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadPathPrimitiveStringValidateDSL = func() {
	Service("ServicePathPrimitiveStringValidate", func() {
		Method("MethodPathPrimitiveStringValidate", func() {
			Payload(String, func() {
				Enum("val")
			})
			HTTP(func() {
				GET("/{p}")
			})
		})
	})
}


