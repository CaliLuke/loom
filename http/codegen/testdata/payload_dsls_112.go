package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyQueryPathUserDSL = func() {
	var PayloadType = Type("PayloadType", func() {
		Attribute("a", String)
		Attribute("b", String)
		Attribute("c", String)
	})
	Service("ServiceBodyQueryPathUser", func() {
		Method("MethodBodyQueryPathUser", func() {
			Payload(PayloadType)
			HTTP(func() {
				POST("/{c}")
				Param("b")
			})
		})
	})
}


