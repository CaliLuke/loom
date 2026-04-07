package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultBodyArrayStringDSL = func() {
	Service("ServiceBodyArrayString", func() {
		Method("MethodBodyArrayString", func() {
			Result(func() {
				Attribute("b", ArrayOf(String))
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}


