package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var PayloadBodyUserInnerDefaultDSL = func() {
	var InnerType = Type("InnerType", func() {
		Attribute("a", String, func() {
			Default("defaulta")
			Pattern("patterna")
		})
		Attribute("b", String, func() {
			Default("defaultb")
			Pattern("patternb")
		})
		Required("a")
	})
	var PayloadType = Type("PayloadType", func() {
		Attribute("inner", InnerType)
	})
	Service("ServiceBodyUserInnerDefault", func() {
		Method("MethodBodyUserInnerDefault", func() {
			Payload(PayloadType)
			HTTP(func() {
				POST("/")
			})
		})
	})
}


