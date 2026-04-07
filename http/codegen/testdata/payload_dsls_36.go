package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadPathPrimitiveStringFormatIPValidateDSL = func() {
	Service("ServicePathPrimitiveStringFormatIPValidate", func() {
		Method("MethodPathPrimitiveStringFormatIPValidate", func() {
			Payload(String, func() {
				Format(FormatIP)
			})
			HTTP(func() {
				GET("/forecast/{ip}")
				Param("ip")
			})
		})
	})
}


