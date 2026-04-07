package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultHeaderStringArrayDSL = func() {
	Service("ServiceHeaderStringArrayResponse", func() {
		Method("MethodA", func() {
			Result(func() {
				Attribute("array", ArrayOf(String))
			})
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("array")
				})
			})
		})
	})
}


