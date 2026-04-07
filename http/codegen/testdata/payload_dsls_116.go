package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyUserOriginDSL = func() {
	var PayloadType = Type("PayloadType", func() {
		Attribute("a")
		Required("a")
	})
	Service("ServiceBodyUserOriginDefault", func() {
		Method("MethodBodyUserOriginDefault", func() {
			Payload(func() {
				Attribute("body", PayloadType)
			})
			HTTP(func() {
				POST("/")
				Body("body")
			})
		})
	})
}


