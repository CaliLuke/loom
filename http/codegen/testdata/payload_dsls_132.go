package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadMultipartMapTypeDSL = func() {
	Service("ServiceMultipartMapType", func() {
		Method("MethodMultipartMapType", func() {
			Payload(MapOf(String, Int))
			HTTP(func() {
				POST("/")
				MultipartRequest()
			})
		})
	})
}


