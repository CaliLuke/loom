package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// Result(Header|Body)(Type)(Required|Default)?DSL
//
// Where Type is the type of the result or result.


var ResultTagStringRequiredDSL = func() {
	Service("ServiceTagStringRequired", func() {
		Method("MethodTagStringRequired", func() {
			Result(func() {
				Attribute("h", String)
				Required("h")
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


