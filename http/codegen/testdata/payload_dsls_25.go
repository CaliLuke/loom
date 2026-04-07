package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadQueryPrimitiveStringDefaultDSL = func() {
	Service("ServiceQueryPrimitiveStringDefault", func() {
		Method("MethodQueryPrimitiveStringDefault", func() {
			Payload(String, func() {
				Default("def")
			})
			HTTP(func() {
				GET("/")
				Param("q")
			})
		})
	})
}


