package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyObjectRequiredDSL = func() {
	Service("ServiceBodyObjectRequired", func() {
		Method("MethodBodyObjectRequired", func() {
			Payload(func() {
				Attribute("b", String)
				Required("b")
			})
			HTTP(func() {
				POST("/")
				Body(func() {
					Attribute("b", String)
					Required("b")
				})
			})
		})
	})
}


