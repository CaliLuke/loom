package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadHeaderStringDSL = func() {
	Service("ServiceHeaderString", func() {
		Method("MethodHeaderString", func() {
			Payload(func() {
				Attribute("h", String)
			})
			HTTP(func() {
				GET("/")
				Header("h")
			})
		})
	})
}


