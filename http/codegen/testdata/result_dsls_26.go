package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultHeaderStringImplicitDSL = func() {
	Service("ServiceHeaderStringImplicit", func() {
		Method("MethodHeaderStringImplicit", func() {
			Result(String)
			HTTP(func() {
				GET("/")
				Response(StatusOK, func() {
					Header("h")
				})
			})
		})
	})
}


