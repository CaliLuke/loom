package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultBodyHeaderObjectDSL = func() {
	Service("ServiceBodyHeaderObject", func() {
		Method("MethodBodyHeaderObject", func() {
			Result(func() {
				Attribute("a", String)
				Attribute("b", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Header("b")
				})
			})
		})
	})
}


