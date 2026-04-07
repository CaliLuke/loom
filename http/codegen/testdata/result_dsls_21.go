package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultTagStringDSL = func() {
	Service("ServiceTagString", func() {
		Method("MethodTagString", func() {
			Result(func() {
				Attribute("h", String)
			})
			HTTP(func() {
				GET("/")
				Response(StatusAccepted, func() {
					Header("h")
					Tag("h", "value")
				})
				Response(StatusOK)
			})
		})
	})
}


