package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultHeaderArrayValidateDSL = func() {
	Service("ServiceHeaderArrayValidateResponse", func() {
		Method("MethodA", func() {
			Result(func() {
				Attribute("array", ArrayOf(Int), func() {
					Elem(func() {
						Minimum(5)
					})
				})
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


