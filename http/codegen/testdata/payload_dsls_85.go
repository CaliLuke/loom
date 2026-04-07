package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyMapUserDSL = func() {
	var PayloadType = Type("PayloadType", func() {
		Attribute("a", String, func() {
			Pattern("apattern")
		})
	})
	Service("ServiceBodyMapUser", func() {
		Method("MethodBodyMapUser", func() {
			Payload(func() {
				Attribute("b", MapOf(String, PayloadType))
			})
			HTTP(func() {
				POST("/")
			})
		})
	})
}


