package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultBodyPrimitiveArrayStringDSL = func() {
	Service("ServiceBodyPrimitiveArrayString", func() {
		Method("MethodBodyPrimitiveArrayString", func() {
			Result(ArrayOf(String), func() {
				MinLength(1)
				Elem(func() {
					Enum("val")
				})
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}


