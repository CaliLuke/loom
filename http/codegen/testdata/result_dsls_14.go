package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultBodyPrimitiveAnyDSL = func() {
	Service("ServiceBodyPrimitiveAny", func() {
		Method("MethodBodyPrimitiveAny", func() {
			Result(Any)
			HTTP(func() {
				POST("/")
			})
		})
	})
}


