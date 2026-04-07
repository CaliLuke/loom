package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadMultipartPrimitiveDSL = func() {
	Service("ServiceMultipartPrimitive", func() {
		Method("MethodMultipartPrimitive", func() {
			Payload(String)
			HTTP(func() {
				POST("/")
				MultipartRequest()
			})
		})
	})
}


