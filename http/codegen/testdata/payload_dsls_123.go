package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadMapQueryPrimitiveArrayDSL = func() {
	Service("ServiceMapQueryPrimitiveArray", func() {
		Method("MapQueryPrimitiveArray", func() {
			Payload(MapOf(String, ArrayOf(UInt)))
			HTTP(func() {
				POST("/")
				MapParams()
			})
		})
	})
}


