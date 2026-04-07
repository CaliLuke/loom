package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyStringDSL = func() {
	Service("ServiceBodyString", func() {
		Method("MethodBodyString", func() {
			Payload(func() {
				Attribute("b", String)
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}


