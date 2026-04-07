package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadHeaderArrayIntDSL = func() {
	Service("ServiceHeaderArrayInt", func() {
		Method("MethodHeaderArrayInt", func() {
			Payload(func() {
				Attribute("h", ArrayOf(Int))
			})
			HTTP(func() {
				GET("/")
				Header("h")
			})
		})
	})
}


