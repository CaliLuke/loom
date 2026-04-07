package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultBodyPrimitiveBoolDSL = func() {
	Service("ServiceBodyPrimitiveBool", func() {
		Method("MethodBodyPrimitiveBool", func() {
			Result(Boolean, func() {
				Enum(true)
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}


