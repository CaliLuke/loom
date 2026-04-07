package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultBodyPrimitiveStringDSL = func() {
	Service("ServiceBodyPrimitiveString", func() {
		Method("MethodBodyPrimitiveString", func() {
			Result(String, func() {
				Enum("val")
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}


