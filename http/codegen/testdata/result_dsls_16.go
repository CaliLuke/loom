package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultBodyPrimitiveArrayBoolDSL = func() {
	Service("ServiceBodyPrimitiveArrayBool", func() {
		Method("MethodBodyPrimitiveArrayBool", func() {
			Result(ArrayOf(Boolean), func() {
				MinLength(1)
				Elem(func() {
					Enum(true)
				})
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}


