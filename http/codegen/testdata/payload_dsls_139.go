package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var MultipleMethodsDSL = func() {
	var APayload = Type("APayload", func() {
		Attribute("a", String, func() {
			Pattern("patterna")
		})
	})
	var PayloadType = Type("PayloadType", func() {
		Attribute("a", String, func() {
			Pattern("patterna")
		})
		Attribute("b", String, func() {
			Pattern("patternb")
		})
		Attribute("c", APayload)
		Required("a", "c")
	})
	Service("ServiceMultipleMethods", func() {
		Method("MethodA", func() {
			Payload(APayload)
			HTTP(func() {
				POST("/")
				Body(APayload)
			})
		})
		Method("MethodB", func() {
			Payload(PayloadType)
			HTTP(func() {
				PUT("/")
			})
		})
	})
}


