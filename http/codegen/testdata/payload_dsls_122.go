package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadMapQueryPrimitivePrimitiveDSL = func() {
	Service("ServiceMapQueryPrimitivePrimitive", func() {
		Method("MapQueryPrimitivePrimitive", func() {
			Payload(MapOf(String, String))
			HTTP(func() {
				POST("/")
				MapParams()
			})
		})
	})
}


