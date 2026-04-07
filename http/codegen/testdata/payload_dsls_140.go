package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// The DSL function names follow the following pattern:
//
// (Payload|Result)(Query|Path|Body)+(Type)(Validate)?DSL
//
// Where Type is the type of the payload or result.


var MultipleMethodsWithArrayTypePayloadsDSL = func() {
	var PayloadA = Type("PayloadA", func() {
		Attribute("a", String, func() {
			Pattern("patterna")
		})
	})
	var PayloadB = Type("PayloadB", func() {
		Attribute("a", String, func() {
			Pattern("patterna")
		})
		Attribute("b", String, func() {
			Pattern("patternb")
		})
		Required("a", "b")
	})
	Service("ServiceMultipleMethods", func() {
		Method("MethodA", func() {
			Payload(ArrayOf(PayloadA))
			HTTP(func() {
				POST("/")
			})
		})
		Method("MethodB", func() {
			Payload(ArrayOf(PayloadB))
			HTTP(func() {
				PUT("/")
			})
		})
	})
}


