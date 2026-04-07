package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadHeaderIntDSL = func() {
	Service("ServiceHeaderInt", func() {
		Method("MethodHeaderInt", func() {
			Payload(func() {
				Attribute("h", Int)
			})
			HTTP(func() {
				GET("/")
				Header("h")
			})
		})
	})
}


