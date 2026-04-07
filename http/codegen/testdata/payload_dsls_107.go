package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyPathUserDSL = func() {
	var PayloadType = Type("PayloadType", func() {
		Attribute("a", String)
		Attribute("b", String)
	})
	Service("ServiceBodyPathUser", func() {
		Method("MethodBodyPathUser", func() {
			Payload(PayloadType)
			HTTP(func() {
				POST("/{b}")
			})
		})
	})
}


